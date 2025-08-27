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

	cerrdefs "github.com/containerd/errdefs"
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
	onDiskLayersMutex    sync.RWMutex

	// containerd storage detection cache
	containerdStorageCache     *bool
	containerdStorageCacheLock sync.RWMutex
}

// DockerClient is subset of client.APIClient required by this package.
type DockerClient interface {
	ImageHistory(ctx context.Context, image string, opts ...client.ImageHistoryOption) ([]image.HistoryResponseItem, error)
	ImageInspect(ctx context.Context, image string, opts ...client.ImageInspectOption) (image.InspectResponse, error)
	ImageLoad(ctx context.Context, input io.Reader, opts ...client.ImageLoadOption) (image.LoadResponse, error)
	ImageRemove(ctx context.Context, image string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	ImageSave(ctx context.Context, images []string, opts ...client.ImageSaveOption) (io.ReadCloser, error)
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
	_, err := s.dockerClient.ImageInspect(context.Background(), identifier)
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

func (s *Store) Save(img *Image, withName string, withAdditionalNames ...string) (string, error) {
	withName = tryNormalizing(withName)
	var (
		inspect image.InspectResponse
		err     error
	)

	// save
	isContainerdStorage := s.usesContainerdStorageCached()

	// During the first save attempt some layers may be excluded.
	// The docker daemon allows this if the given set of layers already exists in the daemon in the given order.
	inspect, err = s.doSave(img, withName, isContainerdStorage)

	// If the fast save fails, we need to ensure the layers and try again.
	if err != nil {
		if err = img.ensureLayers(); err != nil {
			return "", err
		}

		inspect, err = s.doSave(img, withName, isContainerdStorage)
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

// usesContainerdStorageCached provides cached containerd storage detection
func (s *Store) usesContainerdStorageCached() bool {
	s.containerdStorageCacheLock.RLock()
	if s.containerdStorageCache != nil {
		result := *s.containerdStorageCache
		s.containerdStorageCacheLock.RUnlock()
		return result
	}
	s.containerdStorageCacheLock.RUnlock()

	// Need to compute and cache
	s.containerdStorageCacheLock.Lock()
	defer s.containerdStorageCacheLock.Unlock()

	// Double-check after acquiring write lock
	if s.containerdStorageCache != nil {
		return *s.containerdStorageCache
	}

	result := usesContainerdStorage(s.dockerClient)
	s.containerdStorageCache = &result
	return result
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

func (s *Store) doSave(img v1.Image, withName string, isContainerdStorage bool) (image.InspectResponse, error) {
	ctx := context.Background()
	done := make(chan error)

	var err error
	pr, pw := io.Pipe()
	defer pw.Close()

	// Start the ImageLoad goroutine
	go func() {
		res, err := s.dockerClient.ImageLoad(ctx, pr, client.ImageLoadWithQuiet(true))
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

	// Create tar content
	tw := tar.NewWriter(pw)
	defer tw.Close()

	if err = s.addImageToTar(tw, img, withName, isContainerdStorage); err != nil {
		return image.InspectResponse{}, err
	}

	tw.Close()
	pw.Close()

	// Wait for ImageLoad to complete
	err = <-done

	if err != nil {
		return image.InspectResponse{}, fmt.Errorf("loading image %q. first error: %w", withName, err)
	}

	// Inspect the saved image
	inspect, err := s.dockerClient.ImageInspect(context.Background(), withName)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return image.InspectResponse{}, fmt.Errorf("saving image %q: %w", withName, err)
		}
		return image.InspectResponse{}, err
	}

	return inspect, nil
}

func (s *Store) addImageToTar(tw *tar.Writer, image v1.Image, withName string, isContainerdStorage bool) error {
	// Add config file
	rawConfigFile, err := image.RawConfigFile()
	if err != nil {
		return err
	}
	configHash := fmt.Sprintf("%x", sha256.Sum256(rawConfigFile))
	if err = addTextToTar(tw, rawConfigFile, configHash+".json"); err != nil {
		return err
	}

	// Get layers
	layers, err := image.Layers()
	if err != nil {
		return err
	}

	var (
		layerPaths []string
		blankIdx   int
	)
	for _, layer := range layers {
		// If the layer is a previous image layer that hasn't been downloaded yet,
		// cause ALL the previous image layers to be downloaded by grabbing the ReadCloser.
		layerReader, err := layer.Uncompressed()
		if err != nil {
			return err
		}
		defer layerReader.Close()

		var layerName string
		size, err := layer.Size()
		if err != nil {
			return err
		}
		if size == -1 { // it's a base (always empty) layer
			layerName = fmt.Sprintf("blank_%d", blankIdx)
			hdr := &tar.Header{Name: layerName, Mode: 0644, Size: 0}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			// it's a populated layer
			layerDiffID, err := layer.DiffID()
			if err != nil {
				return err
			}
			layerName = fmt.Sprintf("/%s.tar", layerDiffID.String())

			// CONTAINERD OPTIMIZATION: For containerd, calculate size efficiently during tar writing
			var uncompressedSize int64

			if isContainerdStorage {
				// For containerd, use Docker-native format bypass optimization

				// Docker-Native Bypass: Skip custom tar creation, use existing layer files directly

				// Use the layer's existing compressed format if available
				// This mimics what docker save/load does natively
				compressedReader, err := layer.Compressed()
				if err != nil {
					// Fallback to uncompressed if compressed not available
					compressedReader = layerReader
				} else {
					// Close the uncompressed reader since we're using compressed
					layerReader.Close()
				}
				defer compressedReader.Close()

				// Stream compressed data directly to tar (Docker-native format)
				tempFile, err := os.CreateTemp("", "compressed-layer-*.tar")
				if err != nil {
					return err
				}
				defer os.Remove(tempFile.Name())
				defer tempFile.Close()

				// Calculate size while streaming to temp file
				bytesRead, err := io.Copy(tempFile, compressedReader)
				if err != nil {
					return err
				}
				uncompressedSize = bytesRead

				// Rewind temp file for reading
				if _, err := tempFile.Seek(0, 0); err != nil {
					return err
				}

				// Write tar header with calculated size (Docker-native format)
				hdr := &tar.Header{Name: layerName, Mode: 0644, Size: uncompressedSize}
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}

				// Stream from temp file to tar
				_, err = io.Copy(tw, tempFile)
				if err != nil {
					return err
				}
			} else {
				// For standard Docker storage, use original logic
				uncompressedSize, err := s.getLayerSize(layer)
				if err != nil {
					return err
				}
				hdr := &tar.Header{Name: layerName, Mode: 0644, Size: uncompressedSize}
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				if _, err := io.Copy(tw, layerReader); err != nil {
					return err
				}
			}
		}
		blankIdx++
		layerPaths = append(layerPaths, layerName)
	}

	// Add manifest
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
	if err = addTextToTar(tw, manifestJSON, "manifest.json"); err != nil {
		return err
	}

	return nil
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

	s.onDiskLayersMutex.RLock()
	knownLayer, layerFound := s.onDiskLayersByDiffID[diffID]
	s.onDiskLayersMutex.RUnlock()
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

		return s.addImageToTar(tw, image, withName, s.usesContainerdStorageCached())
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

	// Parallel Processing Optimization: Process multiple layers concurrently during download
	// This can significantly reduce wall-clock time, especially for containerd storage where
	// layers are downloaded fresh and need expensive uncompressed size calculations.
	// Use errgroup for parallel layer processing with bounded concurrency
	g, ctx := errgroup.WithContext(context.Background())

	// Limit concurrent layer processing to avoid overwhelming the system
	// Use a reasonable number based on typical layer counts and system resources
	maxConcurrency := 3
	if len(configFile.RootFS.DiffIDs) < maxConcurrency {
		maxConcurrency = len(configFile.RootFS.DiffIDs)
	}

	// Create a channel to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)

	for idx := range configFile.RootFS.DiffIDs {
		idx := idx // capture loop variable
		g.Go(func() error {
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-semaphore }() // Release semaphore

			layerPath := filepath.Join(tmpDir, manifest[0].Layers[idx])

			if _, err := s.AddLayer(layerPath); err != nil {
				return err
			}
			return nil
		})
	}

	// Wait for all layers to complete
	if err := g.Wait(); err != nil {
		return err
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
	s.onDiskLayersMutex.RLock()
	aLayer, layerFound := s.onDiskLayersByDiffID[withHash]
	s.onDiskLayersMutex.RUnlock()
	if !layerFound {
		return nil
	}
	return aLayer.layer
}

// AddLayer adds a layer from a file path to the store's cache.
// This method includes a performance optimization: for compressed layers, it proactively
// calculates the uncompressed size during the add operation to avoid expensive cache misses
// later during tar creation. This optimization is particularly important for containerd storage
// scenarios where layers are downloaded fresh and don't have cached uncompressed sizes.
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
		// the layer is compressed

		// CONTAINERD OPTIMIZATION: Skip expensive size calculation during download for containerd
		// We'll calculate it more efficiently during tar creation
		if s.usesContainerdStorageCached() {
			uncompressedSize = -1 // Will be calculated efficiently during tar creation
		} else {
			// For standard Docker storage, calculate size now to avoid cache misses later

			layerReader, err := layer.Uncompressed()
			if err != nil {
				// Fall back to unknown size to maintain backward compatibility
				uncompressedSize = -1
			} else {
				defer layerReader.Close()
				var calculatedSize int64
				calculatedSize, err = io.Copy(io.Discard, layerReader)
				if err != nil {
					// Fall back to unknown size to maintain backward compatibility
					uncompressedSize = -1
				} else {
					uncompressedSize = calculatedSize
				}
			}
		}
	} else {
		uncompressedSize = fileSize
	}

	s.onDiskLayersMutex.Lock()
	s.onDiskLayersByDiffID[diffID] = annotatedLayer{
		layer:            layer,
		uncompressedSize: uncompressedSize,
	}
	s.onDiskLayersMutex.Unlock()

	return layer, nil
}
