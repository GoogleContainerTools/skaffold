package dist

import "github.com/buildpacks/pack/internal/api"

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

type Order []OrderEntry

type OrderEntry struct {
	Group []BuildpackRef `toml:"group" json:"group"`
}

type BuildpackRef struct {
	BuildpackInfo
	Optional bool `toml:"optional,omitempty" json:"optional,omitempty"`
}

type BuildpackLayers map[string]map[string]BuildpackLayerInfo

type BuildpackLayerInfo struct {
	API         *api.Version `json:"api"`
	Stacks      []Stack      `json:"stacks"`
	Order       Order        `json:"order,omitempty"`
	LayerDiffID string       `json:"layerDiffID"`
}

func AddBuildpackToLayersMD(layerMD BuildpackLayers, descriptor BuildpackDescriptor, diffID string) {
	bpInfo := descriptor.Info
	if _, ok := layerMD[bpInfo.ID]; !ok {
		layerMD[bpInfo.ID] = map[string]BuildpackLayerInfo{}
	}
	layerMD[bpInfo.ID][bpInfo.Version] = BuildpackLayerInfo{
		API:         descriptor.API,
		Stacks:      descriptor.Stacks,
		Order:       descriptor.Order,
		LayerDiffID: diffID,
	}
}
