package dist

import (
	"github.com/buildpacks/lifecycle/api"
)

const (
	BuildpackLayersLabel   = "io.buildpacks.buildpack.layers"
	ExtensionLayersLabel   = "io.buildpacks.extension.layers"
	ExtensionMetadataLabel = "io.buildpacks.extension.metadata"
	DefaultTargetOSLinux   = "linux"
	DefaultTargetOSWindows = "windows"
	DefaultTargetArch      = "amd64"
)

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
	Group []ModuleRef `toml:"group" json:"group"`
}

type ModuleRef struct {
	ModuleInfo `yaml:"buildpackinfo,inline"`
	Optional   bool `toml:"optional,omitempty" json:"optional,omitempty" yaml:"optional,omitempty"`
}

type ModuleLayers map[string]map[string]ModuleLayerInfo

type ModuleLayerInfo struct {
	API         *api.Version `json:"api"`
	Stacks      []Stack      `json:"stacks,omitempty"`
	Targets     []Target     `json:"targets,omitempty"`
	Order       Order        `json:"order,omitempty"`
	LayerDiffID string       `json:"layerDiffID"`
	Homepage    string       `json:"homepage,omitempty"`
	Name        string       `json:"name,omitempty"`
}

func (b ModuleLayers) Get(id, version string) (ModuleLayerInfo, bool) {
	buildpackLayerEntries, ok := b[id]
	if !ok {
		return ModuleLayerInfo{}, false
	}
	if len(buildpackLayerEntries) == 1 && version == "" {
		for key := range buildpackLayerEntries {
			version = key
		}
	}

	result, ok := buildpackLayerEntries[version]
	return result, ok
}
