package fakes

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

type Package interface {
	Name() string
	BuildpackLayers() dist.ModuleLayers
	GetLayer(diffID string) (io.ReadCloser, error)
}

var _ Package = (*fakePackage)(nil)

type fakePackage struct {
	name       string
	bpTarFiles map[string]string
	bpLayers   dist.ModuleLayers
}

func NewPackage(tmpDir string, name string, buildpacks []buildpack.BuildModule) (Package, error) {
	processBuildpack := func(bp buildpack.BuildModule) (tarFile string, diffID string, err error) {
		tarFile, err = buildpack.ToLayerTar(tmpDir, bp)
		if err != nil {
			return "", "", err
		}

		layer, err := tarball.LayerFromFile(tarFile)
		if err != nil {
			return "", "", err
		}

		hash, err := layer.DiffID()
		if err != nil {
			return "", "", err
		}

		return tarFile, hash.String(), nil
	}

	bpLayers := dist.ModuleLayers{}
	bpTarFiles := map[string]string{}
	for _, bp := range buildpacks {
		tarFile, diffID, err := processBuildpack(bp)
		if err != nil {
			return nil, err
		}
		bpTarFiles[diffID] = tarFile
		dist.AddToLayersMD(bpLayers, bp.Descriptor(), diffID)
	}

	return &fakePackage{
		name:       name,
		bpTarFiles: bpTarFiles,
		bpLayers:   bpLayers,
	}, nil
}

func (f *fakePackage) Name() string {
	return f.name
}

func (f *fakePackage) BuildpackLayers() dist.ModuleLayers {
	return f.bpLayers
}

func (f *fakePackage) GetLayer(diffID string) (io.ReadCloser, error) {
	tarFile, ok := f.bpTarFiles[diffID]
	if !ok {
		return nil, errors.New("no layer found")
	}

	return os.Open(filepath.Clean(tarFile))
}
