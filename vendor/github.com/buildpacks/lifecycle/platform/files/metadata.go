package files

import (
	"encoding/json"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
)

// BuildMetadata is written by the builder as <layers>/config/metadata.toml to record information about the build.
// It is also serialized by the exporter as the `io.buildpacks.build.metadata` label on the output image.
type BuildMetadata struct {
	// BOM (deprecated) holds the unstructured bill-of-materials.
	BOM []buildpack.BOMEntry `toml:"bom,omitempty" json:"bom"`
	// Buildpacks are the buildpacks used in the build.
	Buildpacks []buildpack.GroupElement `toml:"buildpacks" json:"buildpacks"`
	// Extensions are the image extensions used in the build.
	Extensions []buildpack.GroupElement `toml:"extensions,omitempty" json:"extensions,omitempty"`
	// Labels are labels provided by buildpacks.
	Labels []buildpack.Label `toml:"labels" json:"-"`
	// Launcher is metadata to describe the launcher.
	Launcher LauncherMetadata `toml:"-" json:"launcher"`
	// Processes are processes provided by buildpacks.
	Processes []launch.Process `toml:"processes" json:"processes"`
	// Slices are application slices provided by buildpacks,
	// used by the exporter to "slice" the application directory into distinct layers.
	Slices []layers.Slice `toml:"slices" json:"-"`
	// BuildpackDefaultProcessType is the buildpack-provided default process type.
	// It will be the default process type for the image unless overridden by the end user.
	BuildpackDefaultProcessType string `toml:"buildpack-default-process-type,omitempty" json:"buildpack-default-process-type,omitempty"`
	// PlatformAPI is the Platform API version used for the build.
	PlatformAPI *api.Version `toml:"-" json:"-"`
}

func (m *BuildMetadata) MarshalJSON() ([]byte, error) {
	if m.PlatformAPI == nil || m.PlatformAPI.LessThan("0.9") {
		return json.Marshal(*m)
	}
	type BuildMetadataSerializer BuildMetadata // prevent infinite recursion when serializing
	return json.Marshal(&struct {
		*BuildMetadataSerializer
		BOM []buildpack.BOMEntry `json:"bom,omitempty"`
	}{
		BuildMetadataSerializer: (*BuildMetadataSerializer)(m),
		BOM:                     []buildpack.BOMEntry{},
	})
}

func (m BuildMetadata) ToLaunchMD() launch.Metadata {
	lmd := launch.Metadata{
		Processes: m.Processes,
	}
	for _, bp := range m.Buildpacks {
		lmd.Buildpacks = append(lmd.Buildpacks, launch.Buildpack{
			API: bp.API,
			ID:  bp.ID,
		})
	}
	return lmd
}

type LauncherMetadata struct {
	Version string         `json:"version"`
	Source  SourceMetadata `json:"source"`
}

type SourceMetadata struct {
	Git GitMetadata `json:"git"`
}

type GitMetadata struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
}
