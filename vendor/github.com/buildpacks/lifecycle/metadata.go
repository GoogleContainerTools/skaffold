package lifecycle

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/launch"
)

const (
	BuildMetadataLabel = "io.buildpacks.build.metadata"
	LayerMetadataLabel = "io.buildpacks.lifecycle.metadata"
	StackIDLabel       = "io.buildpacks.stack.id"
)

type BuildMetadata struct {
	Processes  []launch.Process `toml:"processes" json:"processes"`
	Buildpacks []Buildpack      `toml:"buildpacks" json:"buildpacks"`
	BOM        []BOMEntry       `toml:"bom" json:"bom"`
	Launcher   LauncherMetadata `toml:"-" json:"launcher"`
	Slices     []Slice          `toml:"slices" json:"-"`
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

func (md BuildMetadata) hasProcess(processType string) bool {
	for _, p := range md.Processes {
		if p.Type == processType {
			return true
		}
	}
	return false
}

type CacheMetadata struct {
	Buildpacks []BuildpackLayersMetadata `json:"buildpacks"`
}

func (cm *CacheMetadata) MetadataForBuildpack(id string) BuildpackLayersMetadata {
	for _, bpMD := range cm.Buildpacks {
		if bpMD.ID == id {
			return bpMD
		}
	}
	return BuildpackLayersMetadata{}
}

// NOTE: This struct MUST be kept in sync with `LayersMetadataCompat`
type LayersMetadata struct {
	App        []LayerMetadata           `json:"app" toml:"app"`
	Config     LayerMetadata             `json:"config" toml:"config"`
	Launcher   LayerMetadata             `json:"launcher" toml:"launcher"`
	Buildpacks []BuildpackLayersMetadata `json:"buildpacks" toml:"buildpacks"`
	RunImage   RunImageMetadata          `json:"runImage" toml:"run-image"`
	Stack      StackMetadata             `json:"stack" toml:"stack"`
}

// NOTE: This struct MUST be kept in sync with `LayersMetadata`.
// It exists for situations where the `App` field type cannot be
// guaranteed, yet the original struct data must be maintained.
type LayersMetadataCompat struct {
	App        interface{}               `json:"app" toml:"app"`
	Config     LayerMetadata             `json:"config" toml:"config"`
	Launcher   LayerMetadata             `json:"launcher" toml:"launcher"`
	Buildpacks []BuildpackLayersMetadata `json:"buildpacks" toml:"buildpacks"`
	RunImage   RunImageMetadata          `json:"runImage" toml:"run-image"`
	Stack      StackMetadata             `json:"stack" toml:"stack"`
}

type AnalyzedMetadata struct {
	Image    *ImageIdentifier `toml:"image"`
	Metadata LayersMetadata   `toml:"metadata"`
}

// FIXME: fix key names to be accurate in the daemon case
type ImageIdentifier struct {
	Reference string `toml:"reference"`
}

type LayerMetadata struct {
	SHA string `json:"sha" toml:"sha"`
}

type BuildpackLayersMetadata struct {
	ID      string                            `json:"key" toml:"key"`
	Version string                            `json:"version" toml:"version"`
	Layers  map[string]BuildpackLayerMetadata `json:"layers" toml:"layers"`
	Store   *BuildpackStore                   `json:"store,omitempty" toml:"store"`
}

type BuildpackLayerMetadata struct {
	LayerMetadata
	BuildpackLayerMetadataFile
}

type BuildpackLayerMetadataFile struct {
	Data   interface{} `json:"data" toml:"metadata"`
	Build  bool        `json:"build" toml:"build"`
	Launch bool        `json:"launch" toml:"launch"`
	Cache  bool        `json:"cache" toml:"cache"`
}

type BuildpackStore struct {
	Data map[string]interface{} `json:"metadata" toml:"metadata"`
}

type RunImageMetadata struct {
	TopLayer  string `json:"topLayer" toml:"top-layer"`
	Reference string `json:"reference" toml:"reference"`
}

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

func (m *LayersMetadata) MetadataForBuildpack(id string) BuildpackLayersMetadata {
	for _, bpMD := range m.Buildpacks {
		if bpMD.ID == id {
			return bpMD
		}
	}
	return BuildpackLayersMetadata{}
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
