// Data Format Files for the platform api spec (https://github.com/buildpacks/spec/blob/main/platform.md#data-format).

package platform

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
)

// analyzed.toml

type AnalyzedMetadata struct {
	PreviousImage *ImageIdentifier `toml:"image"`
	Metadata      LayersMetadata   `toml:"metadata"`
	RunImage      *ImageIdentifier `toml:"run-image,omitempty"`
}

// FIXME: fix key names to be accurate in the daemon case
type ImageIdentifier struct {
	Reference string `toml:"reference"`
}

// NOTE: This struct MUST be kept in sync with `LayersMetadataCompat`
type LayersMetadata struct {
	App          []LayerMetadata            `json:"app" toml:"app"`
	BOM          *LayerMetadata             `json:"sbom,omitempty" toml:"sbom,omitempty"`
	Buildpacks   []buildpack.LayersMetadata `json:"buildpacks" toml:"buildpacks"`
	Config       LayerMetadata              `json:"config" toml:"config"`
	Launcher     LayerMetadata              `json:"launcher" toml:"launcher"`
	ProcessTypes LayerMetadata              `json:"process-types" toml:"process-types"`
	RunImage     RunImageMetadata           `json:"runImage" toml:"run-image"`
	Stack        StackMetadata              `json:"stack" toml:"stack"`
}

// NOTE: This struct MUST be kept in sync with `LayersMetadata`.
// It exists for situations where the `App` field type cannot be
// guaranteed, yet the original struct data must be maintained.
type LayersMetadataCompat struct {
	App          interface{}                `json:"app" toml:"app"`
	BOM          *LayerMetadata             `json:"sbom,omitempty" toml:"sbom,omitempty"`
	Buildpacks   []buildpack.LayersMetadata `json:"buildpacks" toml:"buildpacks"`
	Config       LayerMetadata              `json:"config" toml:"config"`
	Launcher     LayerMetadata              `json:"launcher" toml:"launcher"`
	ProcessTypes LayerMetadata              `json:"process-types" toml:"process-types"`
	RunImage     RunImageMetadata           `json:"runImage" toml:"run-image"`
	Stack        StackMetadata              `json:"stack" toml:"stack"`
}

func (m *LayersMetadata) MetadataForBuildpack(id string) buildpack.LayersMetadata {
	for _, bpMD := range m.Buildpacks {
		if bpMD.ID == id {
			return bpMD
		}
	}
	return buildpack.LayersMetadata{}
}

type LayerMetadata struct {
	SHA string `json:"sha" toml:"sha"`
}

type RunImageMetadata struct {
	TopLayer  string `json:"topLayer" toml:"top-layer"`
	Reference string `json:"reference" toml:"reference"`
}

// metadata.toml

type BuildMetadata struct {
	BOM                         []buildpack.BOMEntry       `toml:"bom" json:"bom"`
	Buildpacks                  []buildpack.GroupBuildpack `toml:"buildpacks" json:"buildpacks"`
	Labels                      []buildpack.Label          `toml:"labels" json:"-"`
	Launcher                    LauncherMetadata           `toml:"-" json:"launcher"`
	Processes                   []launch.Process           `toml:"processes" json:"processes"`
	Slices                      []layers.Slice             `toml:"slices" json:"-"`
	BuildpackDefaultProcessType string                     `toml:"buildpack-default-process-type,omitempty" json:"buildpack-default-process-type,omitempty"`
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

func (md BuildMetadata) ToLaunchMD() launch.Metadata {
	lmd := launch.Metadata{
		Processes: md.Processes,
	}
	for _, bp := range md.Buildpacks {
		lmd.Buildpacks = append(lmd.Buildpacks, launch.Buildpack{
			API: bp.API,
			ID:  bp.ID,
		})
	}
	return lmd
}

// plan.toml

type BuildPlan struct {
	Entries []BuildPlanEntry `toml:"entries"`
}

func (p BuildPlan) Find(bpID string) buildpack.Plan {
	var out []buildpack.Require
	for _, entry := range p.Entries {
		for _, provider := range entry.Providers {
			if provider.ID == bpID {
				out = append(out, entry.Requires...)
				break
			}
		}
	}
	return buildpack.Plan{Entries: out}
}

// TODO: ensure at least one claimed entry of each name is provided by the BP
func (p BuildPlan) Filter(metRequires []string) BuildPlan {
	var out []BuildPlanEntry
	for _, planEntry := range p.Entries {
		if !containsEntry(metRequires, planEntry) {
			out = append(out, planEntry)
		}
	}
	return BuildPlan{Entries: out}
}

func containsEntry(metRequires []string, entry BuildPlanEntry) bool {
	for _, met := range metRequires {
		for _, planReq := range entry.Requires {
			if met == planReq.Name {
				return true
			}
		}
	}
	return false
}

type BuildPlanEntry struct {
	Providers []buildpack.GroupBuildpack `toml:"providers"`
	Requires  []buildpack.Require        `toml:"requires"`
}

func (be BuildPlanEntry) NoOpt() BuildPlanEntry {
	var out []buildpack.GroupBuildpack
	for _, p := range be.Providers {
		out = append(out, p.NoOpt().NoAPI().NoHomepage())
	}
	be.Providers = out
	return be
}

// project-metadata.toml

type ProjectMetadata struct {
	Source *ProjectSource `toml:"source" json:"source,omitempty"`
}

type ProjectSource struct {
	Type     string                 `toml:"type" json:"type,omitempty"`
	Version  map[string]interface{} `toml:"version" json:"version,omitempty"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata,omitempty"`
}

// report.toml

type ExportReport struct {
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

// stack.toml

type StackMetadata struct {
	RunImage StackRunImageMetadata `json:"runImage" toml:"run-image"`
}

type StackRunImageMetadata struct {
	Image   string   `toml:"image" json:"image"`
	Mirrors []string `toml:"mirrors" json:"mirrors,omitempty"`
}

func (sm *StackMetadata) BestRunImageMirror(registry string) (string, error) {
	if sm.RunImage.Image == "" {
		return "", errors.New("missing run-image metadata")
	}
	runImageMirrors := []string{sm.RunImage.Image}
	runImageMirrors = append(runImageMirrors, sm.RunImage.Mirrors...)
	runImageRef, err := byRegistry(registry, runImageMirrors)
	if err != nil {
		return "", errors.Wrap(err, "failed to find run-image")
	}
	return runImageRef, nil
}

func byRegistry(reg string, imgs []string) (string, error) {
	if len(imgs) < 1 {
		return "", errors.New("no images provided to search")
	}

	for _, img := range imgs {
		ref, err := name.ParseReference(img, name.WeakValidation)
		if err != nil {
			continue
		}
		if reg == ref.Context().RegistryStr() {
			return img, nil
		}
	}
	return imgs[0], nil
}
