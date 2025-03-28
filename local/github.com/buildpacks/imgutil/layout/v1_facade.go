package layout

import (
	"bytes"
	"fmt"
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

type v1ImageFacade struct {
	v1.Image
	diffIDMap map[v1.Hash]v1.Layer
	digestMap map[v1.Hash]v1.Layer
}

func newImageFacadeFrom(original v1.Image, withMediaTypes imgutil.MediaTypes) (v1.Image, error) {
	configFile, err := original.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	manifestFile, err := original.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	originalLayers, err := original.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to get layers: %w", err)
	}

	ensureLayers := func(idx int, layer v1.Layer) (v1.Layer, error) {
		return newLayerOrFacadeFrom(*configFile, *manifestFile, idx, layer)
	}
	// first, ensure media types
	image, mutated, err := imgutil.EnsureMediaTypesAndLayers(original, withMediaTypes, ensureLayers) // if no media types are requested, this does nothing
	if err != nil {
		return nil, fmt.Errorf("failed to ensure media types: %w", err)
	}
	// then, ensure layers
	if mutated {
		// layers are wrapped in a facade, it is possible to call layer.Compressed or layer.Uncompressed without error
		return image, nil
	}
	// we didn't mutate the image (possibly to preserve the digest), we must wrap the image in a facade
	facade := &v1ImageFacade{
		Image:     original,
		diffIDMap: make(map[v1.Hash]v1.Layer),
		digestMap: make(map[v1.Hash]v1.Layer),
	}
	for idx, l := range originalLayers {
		layer, err := newLayerOrFacadeFrom(*configFile, *manifestFile, idx, l)
		if err != nil {
			return nil, err
		}
		diffID, err := layer.DiffID()
		if err != nil {
			return nil, err
		}
		facade.diffIDMap[diffID] = layer
		digest, err := layer.Digest()
		if err != nil {
			return nil, err
		}
		facade.digestMap[digest] = layer
	}

	return facade, nil
}

func (i *v1ImageFacade) Layers() ([]v1.Layer, error) {
	var layers []v1.Layer
	configFile, err := i.ConfigFile()
	if err != nil {
		return nil, err
	}
	if configFile == nil {
		return nil, nil
	}
	for _, diffID := range configFile.RootFS.DiffIDs {
		l, err := i.LayerByDiffID(diffID)
		if err != nil {
			return nil, err
		}
		layers = append(layers, l)
	}
	return layers, nil
}

func (i *v1ImageFacade) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	if layer, ok := i.diffIDMap[h]; ok {
		return layer, nil
	}
	return nil, fmt.Errorf("failed to find layer with diffID %s", h) // shouldn't get here
}

func (i *v1ImageFacade) LayerByDigest(h v1.Hash) (v1.Layer, error) {
	if layer, ok := i.digestMap[h]; ok {
		return layer, nil
	}
	return nil, fmt.Errorf("failed to find layer with digest %s", h) // shouldn't get here
}

type v1LayerFacade struct {
	v1.Layer
	diffID v1.Hash
	digest v1.Hash
	size   int64
}

func (l *v1LayerFacade) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte{})), nil
}

func (l *v1LayerFacade) DiffID() (v1.Hash, error) {
	return l.diffID, nil
}

func (l *v1LayerFacade) Digest() (v1.Hash, error) {
	return l.digest, nil
}

func (l *v1LayerFacade) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte{})), nil
}

func (l *v1LayerFacade) Size() (int64, error) {
	return l.size, nil
}

func newLayerOrFacadeFrom(configFile v1.ConfigFile, manifestFile v1.Manifest, layerIndex int, originalLayer v1.Layer) (v1.Layer, error) {
	if hasData(originalLayer) {
		return originalLayer, nil
	}
	if layerIndex > len(configFile.RootFS.DiffIDs) {
		return nil, fmt.Errorf("failed to find layer for index %d in config file", layerIndex)
	}
	if layerIndex > (len(manifestFile.Layers)) {
		return nil, fmt.Errorf("failed to find layer for index %d in manifest file", layerIndex)
	}
	return &v1LayerFacade{
		Layer:  originalLayer,
		diffID: configFile.RootFS.DiffIDs[layerIndex],
		digest: manifestFile.Layers[layerIndex].Digest,
		size:   manifestFile.Layers[layerIndex].Size,
	}, nil
}

func hasData(layer v1.Layer) bool {
	if rc, err := layer.Compressed(); err == nil {
		defer rc.Close()
		return true
	}
	return false
}
