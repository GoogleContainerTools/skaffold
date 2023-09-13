package files

import (
	"os"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/log"
)

// Analyzed is written by the analyzer as analyzed.toml and updated in subsequent phases to record information about:
// * the previous image (if it exists),
// * the run image,
// * the build image (if provided).
// The location of the file can be specified by providing `-analyzed <path>` to the lifecycle.
type Analyzed struct {
	// PreviousImage is the build image identifier, if the previous image exists.
	PreviousImage *ImageIdentifier `toml:"image,omitempty"`
	// BuildImage is the build image identifier.
	// It is recorded for use by the restorer in the case that image extensions are used
	// to extend the build image.
	BuildImage *ImageIdentifier `toml:"build-image,omitempty"`
	// LayersMetadata holds information about previously built layers.
	// It is used by the exporter to determine if any layers from the current build are unchanged,
	// to avoid re-uploading the same data to the export target,
	// and to provide information about previously-created layers to buildpacks.
	LayersMetadata LayersMetadata `toml:"metadata"`
	// RunImage holds information about the run image.
	// It is used to validate that buildpacks satisfy os/arch constraints,
	// and to provide information about the export target to buildpacks.
	RunImage *RunImage `toml:"run-image,omitempty"`
}

func ReadAnalyzed(path string, logger log.Logger) (Analyzed, error) {
	var analyzed Analyzed
	if _, err := toml.DecodeFile(path, &analyzed); err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("no analyzed metadata found at path '%s'", path)
			return Analyzed{}, nil
		}
		return Analyzed{}, err
	}
	return analyzed, nil
}

func (a Analyzed) PreviousImageRef() string {
	if a.PreviousImage == nil {
		return ""
	}
	return a.PreviousImage.Reference
}

func (a Analyzed) RunImageImage() string {
	if a.RunImage == nil {
		return ""
	}
	return a.RunImage.Image
}

func (a Analyzed) RunImageRef() string {
	if a.RunImage == nil {
		return ""
	}
	return a.RunImage.Reference
}

func (a Analyzed) RunImageTarget() TargetMetadata {
	if a.RunImage == nil {
		return TargetMetadata{}
	}
	if a.RunImage.TargetMetadata == nil {
		return TargetMetadata{}
	}
	return *a.RunImage.TargetMetadata
}

type ImageIdentifier struct {
	Reference string `toml:"reference"` // FIXME: fix key name to be accurate in the daemon case
}

// NOTE: This struct MUST be kept in sync with `LayersMetadataCompat`
type LayersMetadata struct {
	App          []LayerMetadata            `json:"app" toml:"app"`
	BOM          *LayerMetadata             `json:"sbom,omitempty" toml:"sbom,omitempty"`
	Buildpacks   []buildpack.LayersMetadata `json:"buildpacks" toml:"buildpacks"`
	Config       LayerMetadata              `json:"config" toml:"config"`
	Launcher     LayerMetadata              `json:"launcher" toml:"launcher"`
	ProcessTypes LayerMetadata              `json:"process-types" toml:"process-types"`
	RunImage     RunImageForRebase          `json:"runImage" toml:"run-image"`
	Stack        *Stack                     `json:"stack,omitempty" toml:"stack,omitempty"`
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
	RunImage     RunImageForRebase          `json:"runImage" toml:"run-image"`
	Stack        *Stack                     `json:"stack,omitempty" toml:"stack,omitempty"`
}

func (m *LayersMetadata) LayersMetadataFor(bpID string) buildpack.LayersMetadata {
	for _, bpMD := range m.Buildpacks {
		if bpMD.ID == bpID {
			return bpMD
		}
	}
	return buildpack.LayersMetadata{}
}

type LayerMetadata struct {
	SHA string `json:"sha" toml:"sha"`
}

type RunImageForRebase struct {
	TopLayer  string `json:"topLayer" toml:"top-layer"`
	Reference string `json:"reference" toml:"reference"`
	RunImageForExport
}

// Contains returns true if the provided image reference is found in the existing metadata,
// removing the digest portion of the reference when determining if two image names are equivalent.
func (r *RunImageForRebase) Contains(providedImage string) bool {
	return r.RunImageForExport.Contains(providedImage)
}

func (r *RunImageForRebase) ToStack() Stack {
	return Stack{
		RunImage: RunImageForExport{
			Image:   r.Image,
			Mirrors: r.Mirrors,
		},
	}
}

type RunImage struct {
	Reference string `toml:"reference"`
	// Image specifies the repository name for the image that was provided - either by the platform, or by extensions.
	// When exporting to a daemon, the restorer uses this field to pull the run image if needed for the extender;
	// it can't use `Reference` because this may be a daemon image ID if analyzed.toml was last written by the analyzer.
	Image string `toml:"image,omitempty"`
	// Extend if true indicates that the run image should be extended by the extender.
	Extend         bool            `toml:"extend,omitempty"`
	TargetMetadata *TargetMetadata `json:"target,omitempty" toml:"target,omitempty"`
}

type TargetMetadata struct {
	ID          string `json:"id,omitempty" toml:"id,omitempty"`
	OS          string `json:"os" toml:"os"`
	Arch        string `json:"arch" toml:"arch"`
	ArchVariant string `json:"arch-variant,omitempty" toml:"arch-variant,omitempty"`

	Distro *OSDistro `json:"distro,omitempty" toml:"distro,omitempty"`
}

func (t *TargetMetadata) String() string {
	return encoding.ToJSONMaybe(*t)
}

// OSDistro is the OS distribution that a base image provides.
type OSDistro struct {
	Name    string `json:"name" toml:"name"`
	Version string `json:"version" toml:"version"`
}
