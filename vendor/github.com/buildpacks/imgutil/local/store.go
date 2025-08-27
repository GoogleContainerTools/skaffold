package local

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	registryName "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"golang.org/x/sync/errgroup"

	"github.com/buildpacks/imgutil"
)

// Store provides methods for interacting with a docker daemon
// in order to save, delete, and report the presence of images,
// as well as download layers for a given image.
type Store struct {
	// required
	dockerClient DockerClient
	// optional
	downloadOnce         *sync.Once
	onDiskLayersByDiffID map[v1.Hash]annotatedLayer
}

// DockerClient is subset of client.CommonAPIClient required by this package.
type DockerClient interface {
	ImageHistory(ctx context.Context, image string) ([]image.HistoryResponseItem, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error)
	ImageRemove(ctx context.Context, image string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	ImageSave(ctx context.Context, images []string) (io.ReadCloser, error)
	ImageTag(ctx context.Context, image, ref string) error
	Info(ctx context.Context) (system.Info, error)
	ServerVersion(ctx context.Context) (types.Version, error)
}

type annotatedLayer struct {
	layer            v1.Layer
	uncompressedSize int64
}

func NewStore(dockerClient DockerClient) *Store {
	return &Store{
		dockerClient:         dockerClient,
		downloadOnce:         &sync.Once{},
		onDiskLayersByDiffID: make(map[v1.Hash]annotatedLayer),
	}
}

// images

func (s *Store) Contains(identifier string) bool {
	_, _, err := s.dockerClient.ImageInspectWithRaw(context.Background(), identifier)
	return err == nil
}

func (s *Store) Delete(identifier string) error {
	if !s.Contains(identifier) {
		return nil
	}
	options := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}
	_, err := s.dockerClient.ImageRemove(context.Background(), identifier, options)
	return err
}

func (s *Store) Save(image *Image, withName string, withAdditionalNames ...string) (string, error) {
	withName = tryNormalizing(withName)
	var (
		inspect types.ImageInspect
		err     error
	)

	// save
	canOmitBaseLayers := !usesContainerdStorage(s.dockerClient)
	if canOmitBaseLayers {
		// During the first save attempt some layers may be excluded.
		// The docker daemon allows this if the given set of layers already exists in the daemon in the given order.
		inspect, err = s.doSave(image, withName)
	}
	if !canOmitBaseLayers || err != nil {
		if err = image.ensureLayers(); err != nil {
			return "", err
		}
		inspect, err = s.doSave(image, withName)
		if err != nil {
			saveErr := imgutil.SaveError{}
			for _, n := range append([]string{withName}, withAdditionalNames...) {
				saveErr.Errors = append(saveErr.Errors, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
			}
			return "", saveErr
		}
	}

	// tag additional names
	var errs []imgutil.SaveDiagnostic
	for _, n := range append([]string{withName}, withAdditionalNames...) {
		if err = s.dockerClient.ImageTag(context.Background(), inspect.ID, n); err != nil {
			errs = append(errs, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		}
	}
	if len(errs) > 0 {
		return "", imgutil.SaveError{Errors: errs}
	}

	return inspect.ID, nil
}

func tryNormalizing(name string) string {
	// ensure primary tag is valid
	t, err := registryName.NewTag(name, registryName.WeakValidation)
	if err != nil {
		return name
	}
	return t.Name() // returns valid 'name:tag' appending 'latest', if missing tag
}

func usesContainerdStorage(docker DockerClient) bool {
	info, err := docker.Info(context.Background())
	if err != nil {
		return false
	}

	for _, driverStatus := range info.DriverStatus {
		if driverStatus[0] == "driver-type" && driverStatus[1] == "io.containerd.snapshotter.v1" {
			return true
		}
	}

	return false
}

func (s *Store) doSave(image v1.Image, withName string) (types.ImageInspect, error) {
	ctx := context.Background()
	done := make(chan error)

	var err error
	pr, pw := io.Pipe()
	defer pw.Close()

	go func() {
		var res types.ImageLoadResponse
		res, err = s.dockerClient.ImageLoad(ctx, pr, true)
		if err != nil {
			done <- err
			return
		}

		// only return the response error after the response is drained and closed
		responseErr := checkResponseError(res.Body)
		drainCloseErr := ensureReaderClosed(res.Body)
		if responseErr != nil {
			done <- responseErr
			return
		}
		if drainCloseErr != nil {
			done <- drainCloseErr
		}

		done <- nil
	}()

	tw := tar.NewWriter(pw)
	defer tw.Close()

	if err = s.addImageToTar(tw, image, withName); err != nil {
		return types.ImageInspect{}, err
	}
	tw.Close()
	pw.Close()
	err = <-done
	if err != nil {
		return types.ImageInspect{}, fmt.Errorf("loading image %q. first error: %w", withName, err)
	}

	inspect, _, err := s.dockerClient.ImageInspectWithRaw(context.Background(), withName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return types.ImageInspect{}, fmt.Errorf("saving image %q: %w", withName, err)
		}
		return types.ImageInspect{}, err
	}
	return inspect, nil
}

func (s *Store) addImageToTar(tw *tar.Writer, image v1.Image, withName string) error {
	rawConfigFile, err := image.RawConfigFile()
	if err != nil {
		return err
	}
	configHash := fmt.Sprintf("%x", sha256.Sum256(rawConfigFile))
	if err = addTextToTar(tw, rawConfigFile, configHash+".json"); err != nil {
		return err
	}
	layers, err := image.Layers()
	if err != nil {
		return err
	}
	var (
		layerPaths []string
		blankIdx   int
	)
	for _, layer := range layers {
		layerName, err := s.addLayerToTar(tw, layer, blankIdx)
		if err != nil {
			return err
		}
		blankIdx++
		layerPaths = append(layerPaths, layerName)
	}

	manifestJSON, err := json.Marshal([]map[string]interface{}{
		{
			"Config":   configHash + ".json",
			"RepoTags": []string{withName},
			"Layers":   layerPaths,
		},
	})
	if err != nil {
		return err
	}
	return addTextToTar(tw, manifestJSON, "manifest.json")
}

func (s *Store) addLayerToTar(tw *tar.Writer, layer v1.Layer, blankIdx int) (string, error) {
	// If the layer is a previous image layer that hasn't been downloaded yet,
	// cause ALL the previous image layers to be downloaded by grabbing the ReadCloser.
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return "", err
	}
	defer layerReader.Close()

	var layerName string
	size, err := layer.Size()
	if err != nil {
		return "", err
	}
	if size == -1 { // it's a base (always empty) layer
		layerName = fmt.Sprintf("blank_%d", blankIdx)
		hdr := &tar.Header{Name: layerName, Mode: 0644, Size: 0}
		return layerName, tw.WriteHeader(hdr)
	}
	// it's a populated layer
	layerDiffID, err := layer.DiffID()
	if err != nil {
		return "", err
	}
	layerName = fmt.Sprintf("/%s.tar", layerDiffID.String())

	uncompressedSize, err := s.getLayerSize(layer)
	if err != nil {
		return "", err
	}
	hdr := &tar.Header{Name: layerName, Mode: 0644, Size: uncompressedSize}
	if err = tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	if _, err = io.Copy(tw, layerReader); err != nil {
		return "", err
	}

	return layerName, nil
}

// getLayerSize returns the uncompressed layer size.
// This is needed because the daemon expects uncompressed layer size and a v1.Layer reports compressed layer size;
// in a future where we send OCI layout tars to the daemon we should be able to remove this method
// and the need to track layers individually.
func (s *Store) getLayerSize(layer v1.Layer) (int64, error) {
	diffID, err := layer.DiffID()
	if err != nil {
		return 0, err
	}
	knownLayer, layerFound := s.onDiskLayersByDiffID[diffID]
	if layerFound && knownLayer.uncompressedSize != -1 {
		return knownLayer.uncompressedSize, nil
	}
	// FIXME: this is a time sink and should be avoided if the daemon accepts OCI layout-formatted tars
	// If layer was not seen previously, we need to read it to get the uncompressed size
	// In practice, we should only get here if layers saved from the daemon via `docker save`
	// are output compressed.
	layerReader, err := layer.Uncompressed()
	if err != nil {
		return 0, err
	}
	defer layerReader.Close()

	var size int64
	size, err = io.Copy(io.Discard, layerReader)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func addTextToTar(tw *tar.Writer, fileContents []byte, withName string) error {
	hdr := &tar.Header{Name: withName, Mode: 0644, Size: int64(len(fileContents))}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(fileContents)
	return err
}

func checkResponseError(r io.Reader) error {
	decoder := json.NewDecoder(r)
	var jsonMessage jsonmessage.JSONMessage
	if err := decoder.Decode(&jsonMessage); err != nil {
		return fmt.Errorf("parsing daemon response: %w", err)
	}

	if jsonMessage.Error != nil {
		return fmt.Errorf("embedded daemon response: %w", jsonMessage.Error)
	}
	return nil
}

// ensureReaderClosed drains and closes and reader, returning the first error
func ensureReaderClosed(r io.ReadCloser) error {
	_, err := io.Copy(io.Discard, r)
	if closeErr := r.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}

func (s *Store) SaveFile(image *Image, withName string) (string, error) {
	withName = tryNormalizing(withName)

	f, err := os.CreateTemp("", "imgutil.local.image.export.*.tar")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	// All layers need to be present here. Missing layers are either due to utilization of
	// (1) WithPreviousImage(), or (2) FromBaseImage().
	// The former is only relevant if ReuseLayers() has been called which takes care of resolving them.
	// The latter case needs to be handled explicitly.
	if err = image.ensureLayers(); err != nil {
		return "", err
	}

	errs, _ := errgroup.WithContext(context.Background())
	pr, pw := io.Pipe()

	// File writer
	errs.Go(func() error {
		defer pr.Close()
		_, err = f.ReadFrom(pr)
		return err
	})

	// Tar producer
	errs.Go(func() error {
		defer pw.Close()

		tw := tar.NewWriter(pw)
		defer tw.Close()

		return s.addImageToTar(tw, image, withName)
	})

	err = errs.Wait()
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

// layers

func (s *Store) downloadLayersFor(identifier string) error {
	var err error
	s.downloadOnce.Do(func() {
		err = s.doDownloadLayersFor(identifier)
	})
	return err
}

func (s *Store) doDownloadLayersFor(identifier string) error {
	if identifier == "" {
		return nil
	}
	ctx := context.Background()

	imageReader, err := s.dockerClient.ImageSave(ctx, []string{identifier})
	if err != nil {
		return fmt.Errorf("saving image with ID %q from the docker daemon: %w", identifier, err)
	}
	defer ensureReaderClosed(imageReader)

	tmpDir, err := os.MkdirTemp("", "imgutil.local.image.")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	err = untar(imageReader, tmpDir)
	if err != nil {
		return err
	}

	mf, err := os.Open(filepath.Clean(filepath.Join(tmpDir, "manifest.json")))
	if err != nil {
		return err
	}
	defer mf.Close()

	var manifest []struct {
		Config string
		Layers []string
	}
	if err := json.NewDecoder(mf).Decode(&manifest); err != nil {
		return err
	}
	if len(manifest) != 1 {
		return fmt.Errorf("manifest.json had unexpected number of entries: %d", len(manifest))
	}

	cfg, err := os.Open(filepath.Clean(filepath.Join(tmpDir, manifest[0].Config)))
	if err != nil {
		return err
	}
	defer cfg.Close()
	var configFile struct {
		RootFS struct {
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}
	if err = json.NewDecoder(cfg).Decode(&configFile); err != nil {
		return err
	}

	for idx := range configFile.RootFS.DiffIDs {
		layerPath := filepath.Join(tmpDir, manifest[0].Layers[idx])
		if _, err := s.AddLayer(layerPath); err != nil {
			return err
		}
	}
	return nil
}

func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			return nil
		}
		if err != nil {
			return err
		}

		path, err := cleanPath(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					return err
				}
			}

			fh, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			defer fh.Close()
			_, err = io.Copy(fh, tr) // #nosec G110
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					return err
				}
			}

			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func cleanPath(dest, header string) (string, error) {
	joined := filepath.Join(dest, header)
	if strings.HasPrefix(joined, filepath.Clean(dest)) {
		return joined, nil
	}
	return "", fmt.Errorf("bad filepath: %s", header)
}

func (s *Store) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	layer := s.findLayer(h)
	if layer == nil {
		return nil, fmt.Errorf("failed to find layer with diff ID %q", h.String())
	}
	return layer, nil
}

func (s *Store) findLayer(withHash v1.Hash) v1.Layer {
	aLayer, layerFound := s.onDiskLayersByDiffID[withHash]
	if !layerFound {
		return nil
	}
	return aLayer.layer
}

func (s *Store) AddLayer(fromPath string) (v1.Layer, error) {
	layer, err := tarball.LayerFromFile(fromPath)
	if err != nil {
		return nil, err
	}
	diffID, err := layer.DiffID()
	if err != nil {
		return nil, err
	}
	var uncompressedSize int64
	fileSize, err := func() (int64, error) {
		fi, err := os.Stat(fromPath)
		if err != nil {
			return -1, err
		}
		return fi.Size(), nil
	}()
	if err != nil {
		return nil, err
	}
	compressedSize, err := layer.Size()
	if err != nil {
		return nil, err
	}
	if fileSize == compressedSize {
		// the layer is compressed, we don't know the uncompressed size
		uncompressedSize = -1
	} else {
		uncompressedSize = fileSize
	}
	s.onDiskLayersByDiffID[diffID] = annotatedLayer{
		layer:            layer,
		uncompressedSize: uncompressedSize,
	}
	return layer, nil
}
