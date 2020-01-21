package buildpackage

import (
	"github.com/buildpacks/pack/internal/dist"
)

const MetadataLabel = "io.buildpacks.buildpackage.metadata"

type Config struct {
	Buildpack    dist.BuildpackURI `toml:"buildpack"`
	Dependencies []dist.ImageOrURI `toml:"dependencies"`
}

type Metadata struct {
	dist.BuildpackInfo
	Stacks []dist.Stack `toml:"stacks" json:"stacks"`
}
