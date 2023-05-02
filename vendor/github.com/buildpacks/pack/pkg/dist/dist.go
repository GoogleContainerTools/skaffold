package dist

import (
	"github.com/buildpacks/lifecycle/api"
)

const BuildpackLayersLabel = "io.buildpacks.buildpack.layers"

type BuildpackURI struct {
	URI string `toml:"uri"`
}

type ImageRef struct {
	ImageName string `toml:"image"`
}

type ImageOrURI struct {
	BuildpackURI
	ImageRef
}

func (c *ImageOrURI) DisplayString() string {
	if c.BuildpackURI.URI != "" {
		return c.BuildpackURI.URI
	}

	return c.ImageRef.ImageName
}

type Platform struct {
	OS string `toml:"os"`
}

type Order []OrderEntry

type OrderEntry struct {
	Group []BuildpackRef `toml:"group" json:"group"`
}

type BuildpackRef struct {
	BuildpackInfo `yaml:"buildpackinfo,inline"`
	Optional      bool `toml:"optional,omitempty" json:"optional,omitempty" yaml:"optional,omitempty"`
}

type BuildpackLayers map[string]map[string]BuildpackLayerInfo

type BuildpackLayerInfo struct {
	API         *api.Version `json:"api"`
	Stacks      []Stack      `json:"stacks,omitempty"`
	Order       Order        `json:"order,omitempty"`
	LayerDiffID string       `json:"layerDiffID"`
	Homepage    string       `json:"homepage,omitempty"`
	Name        string       `json:"name,omitempty"`
}

func (b BuildpackLayers) Get(id, version string) (BuildpackLayerInfo, bool) {
	buildpackLayerEntries, ok := b[id]
	if !ok {
		return BuildpackLayerInfo{}, false
	}
	if len(buildpackLayerEntries) == 1 && version == "" {
		for key := range buildpackLayerEntries {
			version = key
		}
	}

	result, ok := buildpackLayerEntries[version]
	return result, ok
}
