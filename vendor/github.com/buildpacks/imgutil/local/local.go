package local

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

type Image struct {
	repoName      string
	docker        client.CommonAPIClient
	inspect       types.ImageInspect
	layerPaths    []string
	downloadOnce  *sync.Once
	prevName      string
	prevImage     *FileSystemLocalImage
	easyAddLayers []string
}

type FileSystemLocalImage struct {
	dir       string
	layersMap map[string]string
}

type ImageOption func(image *Image) (*Image, error)

func WithPreviousImage(imageName string) ImageOption {
	return func(i *Image) (*Image, error) {
		if _, err := inspectOptionalImage(i.docker, imageName); err != nil {
			return i, err
		}

		i.prevName = imageName

		return i, nil
	}
}

func FromBaseImage(imageName string) ImageOption {
	return func(i *Image) (*Image, error) {
		var (
			err     error
			inspect types.ImageInspect
		)

		if inspect, err = inspectOptionalImage(i.docker, imageName); err != nil {
			return i, err
		}

		i.inspect = inspect
		i.layerPaths = make([]string, len(i.inspect.RootFS.Layers))

		return i, nil
	}
}

func NewImage(repoName string, dockerClient client.CommonAPIClient, ops ...ImageOption) (imgutil.Image, error) {
	var err error

	inspect, err := defaultInspect(dockerClient)
	if err != nil {
		return nil, err
	}

	image := &Image{
		docker:       dockerClient,
		repoName:     repoName,
		inspect:      inspect,
		layerPaths:   make([]string, len(inspect.RootFS.Layers)),
		downloadOnce: &sync.Once{},
	}

	for _, v := range ops {
		image, err = v(image)
		if err != nil {
			return nil, err
		}
	}

	return image, nil
}

func (i *Image) Label(key string) (string, error) {
	labels := i.inspect.Config.Labels
	return labels[key], nil
}

func (i *Image) Env(key string) (string, error) {
	for _, envVar := range i.inspect.Config.Env {
		parts := strings.Split(envVar, "=")
		if parts[0] == key {
			return parts[1], nil
		}
	}
	return "", nil
}

func (i *Image) OS() (string, error) {
	return i.inspect.Os, nil
}

func (i *Image) OSVersion() (string, error) {
	return i.inspect.OsVersion, nil
}

func (i *Image) Architecture() (string, error) {
	return i.inspect.Architecture, nil
}

func (i *Image) Rename(name string) {
	i.easyAddLayers = nil
	if prevInspect, _, err := i.docker.ImageInspectWithRaw(context.TODO(), name); err == nil {
		if i.sameBase(prevInspect) {
			i.easyAddLayers = prevInspect.RootFS.Layers[len(i.inspect.RootFS.Layers):]
		}
	}

	i.repoName = name
}

func (i *Image) sameBase(prevInspect types.ImageInspect) bool {
	if len(prevInspect.RootFS.Layers) < len(i.inspect.RootFS.Layers) {
		return false
	}
	for i, baseLayer := range i.inspect.RootFS.Layers {
		if baseLayer != prevInspect.RootFS.Layers[i] {
			return false
		}
	}
	return true
}

func (i *Image) Name() string {
	return i.repoName
}

func (i *Image) Found() bool {
	return i.inspect.ID != ""
}

func (i *Image) Identifier() (imgutil.Identifier, error) {
	return IDIdentifier{
		ImageID: strings.TrimPrefix(i.inspect.ID, "sha256:"),
	}, nil
}

func (i *Image) CreatedAt() (time.Time, error) {
	createdAtTime := i.inspect.Created
	createdTime, err := time.Parse(time.RFC3339Nano, createdAtTime)

	if err != nil {
		return time.Time{}, err
	}
	return createdTime, nil
}

func (i *Image) Rebase(baseTopLayer string, newBase imgutil.Image) error {
	ctx := context.Background()

	// FIND TOP LAYER
	keepLayers := -1
	for idx, diffID := range i.inspect.RootFS.Layers {
		if diffID == baseTopLayer {
			keepLayers = len(i.inspect.RootFS.Layers) - idx - 1
			break
		}
	}
	if keepLayers == -1 {
		return fmt.Errorf("'%s' not found in '%s' during rebase", baseTopLayer, i.repoName)
	}

	// SWITCH BASE LAYERS
	newBaseInspect, _, err := i.docker.ImageInspectWithRaw(ctx, newBase.Name())
	if err != nil {
		return errors.Wrap(err, "analyze read previous image config")
	}
	i.inspect.RootFS.Layers = newBaseInspect.RootFS.Layers
	i.layerPaths = make([]string, len(i.inspect.RootFS.Layers))

	// DOWNLOAD IMAGE
	if err := i.downloadImageOnce(i.repoName); err != nil {
		return err
	}

	// READ MANIFEST.JSON
	b, err := ioutil.ReadFile(filepath.Join(i.prevImage.dir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest []struct{ Layers []string }
	if err := json.Unmarshal(b, &manifest); err != nil {
		return err
	}
	if len(manifest) != 1 {
		return fmt.Errorf("expected 1 image received %d", len(manifest))
	}

	// ADD EXISTING LAYERS
	for _, filename := range manifest[0].Layers[(len(manifest[0].Layers) - keepLayers):] {
		if err := i.AddLayer(filepath.Join(i.prevImage.dir, filename)); err != nil {
			return err
		}
	}

	return nil
}

func (i *Image) SetLabel(key, val string) error {
	if i.inspect.Config.Labels == nil {
		i.inspect.Config.Labels = map[string]string{}
	}

	i.inspect.Config.Labels[key] = val
	return nil
}

func (i *Image) SetEnv(key, val string) error {
	i.inspect.Config.Env = append(i.inspect.Config.Env, fmt.Sprintf("%s=%s", key, val))
	return nil
}

func (i *Image) SetWorkingDir(dir string) error {
	i.inspect.Config.WorkingDir = dir
	return nil
}

func (i *Image) SetEntrypoint(ep ...string) error {
	i.inspect.Config.Entrypoint = ep
	return nil
}

func (i *Image) SetCmd(cmd ...string) error {
	i.inspect.Config.Cmd = cmd
	return nil
}

func (i *Image) TopLayer() (string, error) {
	all := i.inspect.RootFS.Layers

	if len(all) == 0 {
		return "", fmt.Errorf("image '%s' has no layers", i.repoName)
	}

	topLayer := all[len(all)-1]
	return topLayer, nil
}

func (i *Image) GetLayer(diffID string) (io.ReadCloser, error) {
	err := i.downloadImageOnce(i.repoName)
	if err != nil {
		return nil, err
	}

	layerID, ok := i.prevImage.layersMap[diffID]
	if !ok {
		return nil, fmt.Errorf("image '%s' does not contain layer with diff ID '%s'", i.repoName, diffID)
	}
	return os.Open(filepath.Join(i.prevImage.dir, layerID))
}

func (i *Image) AddLayer(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "AddLayer: open layer: %s", path)
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return errors.Wrapf(err, "AddLayer: calculate checksum: %s", path)
	}
	diffID := "sha256:" + hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
	return i.AddLayerWithDiffID(path, diffID)
}

func (i *Image) AddLayerWithDiffID(path, diffID string) error {
	i.inspect.RootFS.Layers = append(i.inspect.RootFS.Layers, diffID)
	i.layerPaths = append(i.layerPaths, path)
	i.easyAddLayers = nil
	return nil
}

func (i *Image) ReuseLayer(diffID string) error {
	if len(i.easyAddLayers) > 0 && i.easyAddLayers[0] == diffID {
		i.inspect.RootFS.Layers = append(i.inspect.RootFS.Layers, diffID)
		i.layerPaths = append(i.layerPaths, "")
		i.easyAddLayers = i.easyAddLayers[1:]
		return nil
	}

	if i.prevName == "" {
		return errors.New("no previous image provided to reuse layers from")
	}

	err := i.downloadImageOnce(i.prevName)
	if err != nil {
		return err
	}

	reuseLayer, ok := i.prevImage.layersMap[diffID]
	if !ok {
		return fmt.Errorf("SHA %s was not found in %s", diffID, i.repoName)
	}

	return i.AddLayer(filepath.Join(i.prevImage.dir, reuseLayer))
}

func (i *Image) Save(additionalNames ...string) error {
	inspect, err := i.doSave()
	if err != nil {
		saveErr := imgutil.SaveError{}
		for _, n := range append([]string{i.Name()}, additionalNames...) {
			saveErr.Errors = append(saveErr.Errors, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		}
		return saveErr
	}
	i.inspect = inspect

	var errs []imgutil.SaveDiagnostic
	for _, n := range append([]string{i.Name()}, additionalNames...) {
		if err := i.docker.ImageTag(context.Background(), i.inspect.ID, n); err != nil {
			errs = append(errs, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		}
	}

	if len(errs) > 0 {
		return imgutil.SaveError{Errors: errs}
	}

	return nil
}

func (i *Image) doSave() (types.ImageInspect, error) {
	ctx := context.Background()
	done := make(chan error)

	t, err := name.NewTag(i.repoName, name.WeakValidation)
	if err != nil {
		return types.ImageInspect{}, err
	}

	//returns valid 'name:tag' appending 'latest', if missing tag
	repoName := t.Name()

	pr, pw := io.Pipe()
	defer pw.Close()
	go func() {
		res, err := i.docker.ImageLoad(ctx, pr, true)
		if err != nil {
			done <- err
			return
		}

		//only return response error after response is drained and closed
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

	configFile, err := i.newConfigFile()
	if err != nil {
		return types.ImageInspect{}, errors.Wrap(err, "generate config file")
	}

	id := fmt.Sprintf("%x", sha256.Sum256(configFile))
	if err := addTextToTar(tw, id+".json", configFile); err != nil {
		return types.ImageInspect{}, err
	}

	var layerPaths []string
	for _, path := range i.layerPaths {
		if path == "" {
			layerPaths = append(layerPaths, "")
			continue
		}
		layerName := fmt.Sprintf("/%x.tar", sha256.Sum256([]byte(path)))
		f, err := os.Open(path)
		if err != nil {
			return types.ImageInspect{}, err
		}
		defer f.Close()
		if err := addFileToTar(tw, layerName, f); err != nil {
			return types.ImageInspect{}, err
		}
		f.Close()
		layerPaths = append(layerPaths, layerName)
	}

	manifest, err := json.Marshal([]map[string]interface{}{
		{
			"Config":   id + ".json",
			"RepoTags": []string{repoName},
			"Layers":   layerPaths,
		},
	})
	if err != nil {
		return types.ImageInspect{}, err
	}

	if err := addTextToTar(tw, "manifest.json", manifest); err != nil {
		return types.ImageInspect{}, err
	}

	tw.Close()
	pw.Close()
	err = <-done
	if err != nil {
		return types.ImageInspect{}, errors.Wrapf(err, "image load '%s'. first error", i.repoName)
	}

	inspect, _, err := i.docker.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		if client.IsErrNotFound(err) {
			return types.ImageInspect{}, errors.Wrapf(err, "save image '%s'", i.repoName)
		}
		return types.ImageInspect{}, err
	}

	return inspect, nil
}

func (i *Image) newConfigFile() ([]byte, error) {
	cfg, err := v1Config(i.inspect)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cfg)
}

func (i *Image) Delete() error {
	if !i.Found() {
		return nil
	}
	options := types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	}
	_, err := i.docker.ImageRemove(context.Background(), i.inspect.ID, options)
	return err
}

func (i *Image) downloadImageOnce(imageName string) error {
	var err error
	i.downloadOnce.Do(func() {
		var fsimg *FileSystemLocalImage
		fsimg, err = downloadImage(i.docker, imageName)
		i.prevImage = fsimg
	})
	return err
}

func downloadImage(docker client.CommonAPIClient, imageName string) (*FileSystemLocalImage, error) {
	ctx := context.Background()

	imageReader, err := docker.ImageSave(ctx, []string{imageName})
	if err != nil {
		return nil, err
	}
	defer ensureReaderClosed(imageReader)

	tmpDir, err := ioutil.TempDir("", "imgutil.local.image.")
	if err != nil {
		return nil, errors.Wrap(err, "local reuse-layer create temp dir")
	}

	err = untar(imageReader, tmpDir)
	if err != nil {
		return nil, err
	}

	mf, err := os.Open(filepath.Join(tmpDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	defer mf.Close()

	var manifest []struct {
		Config string
		Layers []string
	}
	if err := json.NewDecoder(mf).Decode(&manifest); err != nil {
		return nil, err
	}

	if len(manifest) != 1 {
		return nil, fmt.Errorf("manifest.json had unexpected number of entries: %d", len(manifest))
	}

	df, err := os.Open(filepath.Join(tmpDir, manifest[0].Config))
	if err != nil {
		return nil, err
	}
	defer df.Close()

	var details struct {
		RootFS struct {
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}

	if err = json.NewDecoder(df).Decode(&details); err != nil {
		return nil, err
	}

	if len(manifest[0].Layers) != len(details.RootFS.DiffIDs) {
		return nil, fmt.Errorf("layers and diff IDs do not match, there are %d layers and %d diffIDs", len(manifest[0].Layers), len(details.RootFS.DiffIDs))
	}

	layersMap := make(map[string]string, len(manifest[0].Layers))
	for i, diffID := range details.RootFS.DiffIDs {
		layerID := manifest[0].Layers[i]
		layersMap[diffID] = layerID
	}

	return &FileSystemLocalImage{
		dir:       tmpDir,
		layersMap: layersMap,
	}, nil
}

func addTextToTar(tw *tar.Writer, name string, contents []byte) error {
	hdr := &tar.Header{Name: name, Mode: 0644, Size: int64(len(contents))}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(contents)
	return err
}

func addFileToTar(tw *tar.Writer, name string, contents *os.File) error {
	fi, err := contents.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{Name: name, Mode: 0644, Size: fi.Size()}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, contents)
	return err
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

		path := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					return err
				}
			}

			fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(fh, tr); err != nil {
				fh.Close()
				return err
			}
			fh.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func inspectOptionalImage(docker client.CommonAPIClient, imageName string) (types.ImageInspect, error) {
	var (
		err     error
		inspect types.ImageInspect
	)

	if inspect, _, err = docker.ImageInspectWithRaw(context.Background(), imageName); err != nil {
		if client.IsErrNotFound(err) {
			return defaultInspect(docker)
		}

		return types.ImageInspect{}, errors.Wrapf(err, "verifying image '%s'", imageName)
	}

	return inspect, nil
}

func defaultInspect(docker client.CommonAPIClient) (types.ImageInspect, error) {
	daemonInfo, err := docker.Info(context.Background())
	if err != nil {
		return types.ImageInspect{}, err
	}

	return types.ImageInspect{
		Os:           daemonInfo.OSType,
		OsVersion:    daemonInfo.OSVersion,
		Architecture: "amd64",
		Config:       &container.Config{},
	}, nil
}

func v1Config(inspect types.ImageInspect) (v1.ConfigFile, error) {
	history := make([]v1.History, len(inspect.RootFS.Layers))
	for i := range history {
		// zero history
		history[i] = v1.History{
			Created: v1.Time{Time: imgutil.NormalizedDateTime},
		}
	}
	diffIDs := make([]v1.Hash, len(inspect.RootFS.Layers))
	for i, layer := range inspect.RootFS.Layers {
		hash, err := v1.NewHash(layer)
		if err != nil {
			return v1.ConfigFile{}, err
		}
		diffIDs[i] = hash
	}
	exposedPorts := make(map[string]struct{}, len(inspect.Config.ExposedPorts))
	for key, val := range inspect.Config.ExposedPorts {
		exposedPorts[string(key)] = val
	}
	var config v1.Config
	if inspect.Config != nil {
		var healthcheck *v1.HealthConfig
		if inspect.Config.Healthcheck != nil {
			healthcheck = &v1.HealthConfig{
				Test:        inspect.Config.Healthcheck.Test,
				Interval:    inspect.Config.Healthcheck.Interval,
				Timeout:     inspect.Config.Healthcheck.Timeout,
				StartPeriod: inspect.Config.Healthcheck.StartPeriod,
				Retries:     inspect.Config.Healthcheck.Retries,
			}
		}
		config = v1.Config{
			AttachStderr:    inspect.Config.AttachStderr,
			AttachStdin:     inspect.Config.AttachStdin,
			AttachStdout:    inspect.Config.AttachStdout,
			Cmd:             inspect.Config.Cmd,
			Healthcheck:     healthcheck,
			Domainname:      inspect.Config.Domainname,
			Entrypoint:      inspect.Config.Entrypoint,
			Env:             inspect.Config.Env,
			Hostname:        inspect.Config.Hostname,
			Image:           inspect.Config.Image,
			Labels:          inspect.Config.Labels,
			OnBuild:         inspect.Config.OnBuild,
			OpenStdin:       inspect.Config.OpenStdin,
			StdinOnce:       inspect.Config.StdinOnce,
			Tty:             inspect.Config.Tty,
			User:            inspect.Config.User,
			Volumes:         inspect.Config.Volumes,
			WorkingDir:      inspect.Config.WorkingDir,
			ExposedPorts:    exposedPorts,
			ArgsEscaped:     inspect.Config.ArgsEscaped,
			NetworkDisabled: inspect.Config.NetworkDisabled,
			MacAddress:      inspect.Config.MacAddress,
			StopSignal:      inspect.Config.StopSignal,
			Shell:           inspect.Config.Shell,
		}
	}
	return v1.ConfigFile{
		Architecture: inspect.Architecture,
		Created:      v1.Time{Time: imgutil.NormalizedDateTime},
		History:      history,
		OS:           inspect.Os,
		OSVersion:    inspect.OsVersion,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: diffIDs,
		},
		Config: config,
	}, nil
}

func checkResponseError(r io.Reader) error {
	decoder := json.NewDecoder(r)
	var jsonMessage jsonmessage.JSONMessage
	if err := decoder.Decode(&jsonMessage); err != nil {
		return errors.Wrapf(err, "parsing daemon response")
	}

	if jsonMessage.Error != nil {
		return errors.Wrap(jsonMessage.Error, "embedded daemon response")
	}
	return nil
}

// ensureReaderClosed drains and closes and reader, returning the first error
func ensureReaderClosed(r io.ReadCloser) error {
	_, err := io.Copy(ioutil.Discard, r)
	if closeErr := r.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}
