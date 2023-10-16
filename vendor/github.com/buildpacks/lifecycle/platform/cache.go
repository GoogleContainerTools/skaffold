package platform

import (
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/platform/files"
)

type CacheMetadata struct {
	BOM        files.LayerMetadata        `json:"sbom"`
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
