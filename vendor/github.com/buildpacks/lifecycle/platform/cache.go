package platform

import "github.com/buildpacks/lifecycle/buildpack"

type CacheMetadata struct {
	BOM        LayerMetadata              `json:"sbom"`
	Buildpacks []buildpack.LayersMetadata `json:"buildpacks"`
}

func (cm *CacheMetadata) MetadataForBuildpack(id string) buildpack.LayersMetadata {
	for _, bpMD := range cm.Buildpacks {
		if bpMD.ID == id {
			return bpMD
		}
	}
	return buildpack.LayersMetadata{}
}
