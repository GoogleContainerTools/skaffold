package dist

import (
	"os"
	"path/filepath"

	"github.com/buildpacks/lifecycle/api"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

type Descriptor interface {
	API() *api.Version
	Info() ModuleInfo
	Order() Order
	Stacks() []Stack
	Targets() []Target
}

func LayerDiffID(layerTarPath string) (v1.Hash, error) {
	fh, err := os.Open(filepath.Clean(layerTarPath))
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "opening tar file")
	}
	defer fh.Close()

	layer, err := tarball.LayerFromFile(layerTarPath)
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "reading layer tar")
	}

	hash, err := layer.DiffID()
	if err != nil {
		return v1.Hash{}, errors.Wrap(err, "generating diff id")
	}

	return hash, nil
}

func AddToLayersMD(layerMD ModuleLayers, descriptor Descriptor, diffID string) {
	info := descriptor.Info()
	if _, ok := layerMD[info.ID]; !ok {
		layerMD[info.ID] = map[string]ModuleLayerInfo{}
	}
	layerMD[info.ID][info.Version] = ModuleLayerInfo{
		API:         descriptor.API(),
		Stacks:      descriptor.Stacks(),
		Targets:     descriptor.Targets(),
		Order:       descriptor.Order(),
		LayerDiffID: diffID,
		Homepage:    info.Homepage,
		Name:        info.Name,
	}
}
