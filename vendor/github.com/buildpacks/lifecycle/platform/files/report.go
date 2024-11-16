package files

import "github.com/buildpacks/lifecycle/buildpack"

// Report is written by the exporter to record information about the build.
// It is not included in the output image, but can be saved off by the platform before the build container exits.
// The location of the file can be specified by providing `-report <path>` to the lifecycle.
type Report struct {
	Build BuildReport `toml:"build,omitempty"`
	Image ImageReport `toml:"image"`
}

type BuildReport struct {
	BOM []buildpack.BOMEntry `toml:"bom"`
}

type ImageReport struct {
	Tags         []string `toml:"tags"`
	ImageID      string   `toml:"image-id,omitempty"`
	Digest       string   `toml:"digest,omitempty"`
	ManifestSize int64    `toml:"manifest-size,omitzero"`
}

// RebaseReport is written by the rebaser to record information about the rebased image.
type RebaseReport struct {
	Image ImageReport `toml:"image"`
}
