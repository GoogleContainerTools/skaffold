package local

import (
	"errors"
	"fmt"
	"io"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

// Image wraps an imgutil.CNBImageCore and implements the methods needed to complete the imgutil.Image interface.
type Image struct {
	*imgutil.CNBImageCore
	repoName       string
	store          *Store
	lastIdentifier string
	daemonOS       string
}

func (i *Image) Kind() string {
	return "local"
}

func (i *Image) Name() string {
	return i.repoName
}

func (i *Image) Rename(name string) {
	i.repoName = name
}

func (i *Image) Found() bool {
	return i.lastIdentifier != ""
}

func (i *Image) Identifier() (imgutil.Identifier, error) {
	return IDIdentifier{
		ImageID: strings.TrimPrefix(i.lastIdentifier, "sha256:"),
	}, nil
}

func (i *Image) Valid() bool {
	// local images are actually always invalid, because they are missing a manifest, but let's ignore that for now
	return true
}

// GetLayer returns an io.ReadCloser with uncompressed layer data.
// The layer will always have data, even if that means downloading ALL the image layers from the daemon.
func (i *Image) GetLayer(diffID string) (io.ReadCloser, error) {
	layerHash, err := v1.NewHash(diffID)
	if err != nil {
		return nil, err
	}
	configFile, err := i.ConfigFile()
	if err != nil {
		return nil, err
	}
	if !contains(configFile.RootFS.DiffIDs, layerHash) {
		return nil, imgutil.ErrLayerNotFound{DiffID: layerHash.String()}
	}
	layer, err := i.LayerByDiffID(layerHash)
	if err == nil {
		// this avoids downloading ALL the image layers from the daemon
		// if the layer is available locally
		// (e.g., it was added using AddLayer).
		if size, err := layer.Size(); err != nil && size != -1 {
			return layer.Uncompressed()
		}
	}
	if err = i.ensureLayers(); err != nil {
		return nil, err
	}
	layer, err = i.LayerByDiffID(layerHash)
	if err != nil {
		return nil, err
	}
	return layer.Uncompressed()
}

func contains(diffIDs []v1.Hash, hash v1.Hash) bool {
	for _, diffID := range diffIDs {
		if diffID.String() == hash.String() {
			return true
		}
	}
	return false
}

func (i *Image) ensureLayers() error {
	if err := i.store.downloadLayersFor(i.lastIdentifier); err != nil {
		return fmt.Errorf("failed to fetch base layers: %w", err)
	}
	return nil
}

func (i *Image) SetOS(osVal string) error {
	if osVal != i.daemonOS {
		return errors.New("invalid os: must match the daemon")
	}
	return i.CNBImageCore.SetOS(osVal)
}

var emptyHistory = v1.History{Created: v1.Time{Time: imgutil.NormalizedDateTime}}

func (i *Image) AddLayer(path string) error {
	layer, err := i.store.AddLayer(path)
	if err != nil {
		return err
	}
	return i.AddLayerWithHistory(layer, emptyHistory)
}

func (i *Image) AddLayerWithDiffID(path, _ string) error {
	layer, err := i.store.AddLayer(path)
	if err != nil {
		return err
	}
	return i.AddLayerWithHistory(layer, emptyHistory)
}

func (i *Image) AddLayerWithDiffIDAndHistory(path, _ string, history v1.History) error {
	layer, err := i.store.AddLayer(path)
	if err != nil {
		return err
	}
	return i.AddLayerWithHistory(layer, history)
}

func (i *Image) AddOrReuseLayerWithHistory(path string, diffID string, history v1.History) error {
	prevLayerExists, err := i.PreviousImageHasLayer(diffID)
	if err != nil {
		return err
	}
	if !prevLayerExists {
		return i.AddLayerWithDiffIDAndHistory(path, diffID, history)
	}
	return i.ReuseLayerWithHistory(diffID, history)
}

func (i *Image) Rebase(baseTopLayerDiffID string, withNewBase imgutil.Image) error {
	if err := i.ensureLayers(); err != nil {
		return err
	}
	return i.CNBImageCore.Rebase(baseTopLayerDiffID, withNewBase)
}

func (i *Image) Save(additionalNames ...string) error {
	err := i.SetCreatedAtAndHistory()
	if err != nil {
		return err
	}
	i.lastIdentifier, err = i.store.Save(i, i.Name(), additionalNames...)
	return err
}

func (i *Image) SaveAs(name string, additionalNames ...string) error {
	err := i.SetCreatedAtAndHistory()
	if err != nil {
		return err
	}
	i.lastIdentifier, err = i.store.Save(i, name, additionalNames...)
	return err
}

func (i *Image) SaveFile() (string, error) {
	return i.store.SaveFile(i, i.Name())
}

func (i *Image) Delete() error {
	return i.store.Delete(i.lastIdentifier)
}
