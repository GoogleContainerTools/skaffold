package files

import (
	iname "github.com/buildpacks/lifecycle/internal/name"
)

// Stack (deprecated as of Platform API 0.12) is provided by the platform as stack.toml to record information about the run images
// that may be used during export.
// It is also serialized by the exporter as the `stack` key in the `io.buildpacks.lifecycle.metadata` label on the output image
// for use during rebase.
// The location of the file can be specified by providing `-stack <path>` to the lifecycle.
type Stack struct {
	RunImage RunImageForExport `json:"runImage" toml:"run-image"`
}

type RunImageForExport struct {
	Image   string   `toml:"image,omitempty" json:"image,omitempty"`
	Mirrors []string `toml:"mirrors,omitempty" json:"mirrors,omitempty"`
}

// Contains returns true if the provided image reference is found in the existing metadata,
// removing the digest portion of the reference when determining if two image names are equivalent.
func (r *RunImageForExport) Contains(providedImage string) bool {
	providedImage = iname.ParseMaybe(providedImage)
	if iname.ParseMaybe(r.Image) == providedImage {
		return true
	}
	for _, m := range r.Mirrors {
		if iname.ParseMaybe(m) == providedImage {
			return true
		}
	}
	return false
}
