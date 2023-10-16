package buildpack

import (
	"github.com/buildpacks/pack/pkg/dist"
)

// TODO: Move to dist
const MetadataLabel = "io.buildpacks.buildpackage.metadata"

type Metadata struct {
	dist.ModuleInfo
	Stacks []dist.Stack `toml:"stacks" json:"stacks"`
}
