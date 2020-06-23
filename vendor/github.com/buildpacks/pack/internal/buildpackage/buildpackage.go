package buildpackage

import (
	"github.com/buildpacks/pack/internal/dist"
)

const MetadataLabel = "io.buildpacks.buildpackage.metadata"

type Metadata struct {
	dist.BuildpackInfo
	Stacks []dist.Stack `toml:"stacks" json:"stacks"`
}
