package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

type Image struct {
	docker           DockerClient
	repoName         string
	inspect          types.ImageInspect
	history          []v1.History
	layerPaths       []string
	prevImage        *Image // reused layers will be fetched from prevImage
	downloadBaseOnce *sync.Once
	createdAt        time.Time
	withHistory      bool
}

// DockerClient is subset of client.CommonAPIClient required by this package
type DockerClient interface {
	ImageHistory(ctx context.Context, image string) ([]image.HistoryResponseItem, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error)
	ImageRemove(ctx context.Context, image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageSave(ctx context.Context, images []string) (io.ReadCloser, error)
	ImageTag(ctx context.Context, image, ref string) error
	Info(ctx context.Context) (types.Info, error)
}

// getters

func (i *Image) Architecture() (string, error) {
	return i.inspect.Architecture, nil
}

func (i *Image) CreatedAt() (time.Time, error) {
	createdAtTime := i.inspect.Created
	createdTime, err := time.Parse(time.RFC3339Nano, createdAtTime)

	if err != nil {
		return time.Time{}, err
	}
	return createdTime, nil
}

func (i *Image) Entrypoint() ([]string, error) {
	return i.inspect.Config.Entrypoint, nil
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

func (i *Image) Found() bool {
	return i.inspect.ID != ""
}

func (i *Image) Valid() bool {
	return i.Found()
}

func (i *Image) GetAnnotateRefName() (string, error) {
	return "", nil
}

func (i *Image) GetLayer(diffID string) (io.ReadCloser, error) {
	for l := range i.inspect.RootFS.Layers {
		if i.inspect.RootFS.Layers[l] != diffID {
			continue
		}
		if i.layerPaths[l] == "" {
			if err := i.downloadBaseLayersOnce(); err != nil {
				return nil, err
			}
			if i.layerPaths[l] == "" {
				return nil, fmt.Errorf("fetching layer %q from daemon", diffID)
			}
		}
		return os.Open(i.layerPaths[l])
	}

	return nil, fmt.Errorf("image %q does not contain layer with diff ID %q", i.repoName, diffID)
}

func (i *Image) History() ([]v1.History, error) {
	return i.history, nil
}

func (i *Image) Identifier() (imgutil.Identifier, error) {
	return IDIdentifier{
		ImageID: strings.TrimPrefix(i.inspect.ID, "sha256:"),
	}, nil
}

func (i *Image) Label(key string) (string, error) {
	labels := i.inspect.Config.Labels
	return labels[key], nil
}

func (i *Image) Labels() (map[string]string, error) {
	copiedLabels := make(map[string]string)
	for i, l := range i.inspect.Config.Labels {
		copiedLabels[i] = l
	}
	return copiedLabels, nil
}

func (i *Image) ManifestSize() (int64, error) {
	return 0, nil
}

func (i *Image) Name() string {
	return i.repoName
}

func (i *Image) OS() (string, error) {
	return i.inspect.Os, nil
}

func (i *Image) OSVersion() (string, error) {
	return i.inspect.OsVersion, nil
}

func (i *Image) TopLayer() (string, error) {
	all := i.inspect.RootFS.Layers

	if len(all) == 0 {
		return "", fmt.Errorf("image %q has no layers", i.repoName)
	}

	topLayer := all[len(all)-1]
	return topLayer, nil
}

func (i *Image) Variant() (string, error) {
	return i.inspect.Variant, nil
}

func (i *Image) WorkingDir() (string, error) {
	return i.inspect.Config.WorkingDir, nil
}

// setters

func (i *Image) AnnotateRefName(refName string) error {
	return nil
}

func (i *Image) Rename(name string) {
	i.repoName = name
}

func (i *Image) SetArchitecture(architecture string) error {
	i.inspect.Architecture = architecture
	return nil
}

func (i *Image) SetCmd(cmd ...string) error {
	i.inspect.Config.Cmd = cmd
	return nil
}

func (i *Image) SetEntrypoint(ep ...string) error {
	i.inspect.Config.Entrypoint = ep
	return nil
}

func (i *Image) SetEnv(key, val string) error {
	ignoreCase := i.inspect.Os == "windows"
	for idx, kv := range i.inspect.Config.Env {
		parts := strings.SplitN(kv, "=", 2)
		foundKey := parts[0]
		searchKey := key
		if ignoreCase {
			foundKey = strings.ToUpper(foundKey)
			searchKey = strings.ToUpper(searchKey)
		}
		if foundKey == searchKey {
			i.inspect.Config.Env[idx] = fmt.Sprintf("%s=%s", key, val)
			return nil
		}
	}
	i.inspect.Config.Env = append(i.inspect.Config.Env, fmt.Sprintf("%s=%s", key, val))
	return nil
}

func (i *Image) SetHistory(history []v1.History) error {
	i.history = history
	return nil
}

func (i *Image) SetLabel(key, val string) error {
	if i.inspect.Config.Labels == nil {
		i.inspect.Config.Labels = map[string]string{}
	}

	i.inspect.Config.Labels[key] = val
	return nil
}

func (i *Image) SetOS(osVal string) error {
	if osVal != i.inspect.Os {
		return fmt.Errorf("invalid os: must match the daemon: %q", i.inspect.Os)
	}
	return nil
}

func (i *Image) SetOSVersion(osVersion string) error {
	i.inspect.OsVersion = osVersion
	return nil
}

func (i *Image) SetVariant(v string) error {
	i.inspect.Variant = v
	return nil
}

func (i *Image) SetWorkingDir(dir string) error {
	i.inspect.Config.WorkingDir = dir
	return nil
}

// modifiers

func (i *Image) AddLayer(path string) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return errors.Wrapf(err, "AddLayer: open layer: %s", path)
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return errors.Wrapf(err, "AddLayer: calculate checksum: %s", path)
	}
	diffID := "sha256:" + hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
	return i.AddLayerWithDiffIDAndHistory(path, diffID, v1.History{})
}

func (i *Image) AddLayerWithDiffID(path, diffID string) error {
	return i.AddLayerWithDiffIDAndHistory(path, diffID, v1.History{})
}

func (i *Image) AddLayerWithDiffIDAndHistory(path, diffID string, history v1.History) error {
	i.layerPaths = append(i.layerPaths, path)
	i.inspect.RootFS.Layers = append(i.inspect.RootFS.Layers, diffID)
	i.history = append(i.history, history)
	return nil
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

func (i *Image) Rebase(baseTopLayer string, newBase imgutil.Image) error {
	ctx := context.Background()

	// FIND TOP LAYER
	var keepLayersIdx int
	for idx, diffID := range i.inspect.RootFS.Layers {
		if diffID == baseTopLayer {
			keepLayersIdx = idx + 1
			break
		}
	}
	if keepLayersIdx == 0 {
		return fmt.Errorf("%q not found in %q during rebase", baseTopLayer, i.repoName)
	}

	// DOWNLOAD IMAGE
	if err := i.downloadBaseLayersOnce(); err != nil {
		return err
	}

	// SWITCH BASE LAYERS
	newBaseInspect, _, err := i.docker.ImageInspectWithRaw(ctx, newBase.Name())
	if err != nil {
		return errors.Wrapf(err, "read config for new base image %q", newBase)
	}
	i.inspect.ID = newBaseInspect.ID
	i.downloadBaseOnce = &sync.Once{}
	i.inspect.RootFS.Layers = append(newBaseInspect.RootFS.Layers, i.inspect.RootFS.Layers[keepLayersIdx:]...)
	i.layerPaths = append(make([]string, len(newBaseInspect.RootFS.Layers)), i.layerPaths[keepLayersIdx:]...)
	return nil
}

func (i *Image) RemoveLabel(key string) error {
	delete(i.inspect.Config.Labels, key)
	return nil
}

func (i *Image) ReuseLayer(diffID string) error {
	if err := i.ensureLayers(); err != nil {
		return err
	}
	for idx := range i.prevImage.inspect.RootFS.Layers {
		if i.prevImage.inspect.RootFS.Layers[idx] == diffID {
			return i.AddLayerWithDiffIDAndHistory(i.prevImage.layerPaths[idx], diffID, i.prevImage.history[idx])
		}
	}
	return fmt.Errorf("SHA %s was not found in %s", diffID, i.prevImage.Name())
}

func (i *Image) ReuseLayerWithHistory(diffID string, history v1.History) error {
	if err := i.ensureLayers(); err != nil {
		return err
	}
	for idx := range i.prevImage.inspect.RootFS.Layers {
		if i.prevImage.inspect.RootFS.Layers[idx] == diffID {
			return i.AddLayerWithDiffIDAndHistory(i.prevImage.layerPaths[idx], diffID, history)
		}
	}
	return fmt.Errorf("SHA %s was not found in %s", diffID, i.prevImage.Name())
}

func (i *Image) ensureLayers() error {
	if i.prevImage == nil {
		return errors.New("failed to reuse layer because no previous image was provided")
	}
	if !i.prevImage.Found() {
		return fmt.Errorf("failed to reuse layer because previous image %q was not found in daemon", i.prevImage.repoName)
	}
	if err := i.prevImage.downloadBaseLayersOnce(); err != nil {
		return err
	}
	return nil
}
