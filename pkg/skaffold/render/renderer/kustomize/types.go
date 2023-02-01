package kustomize

type Kustomization struct {
	// PatchStrategicMerge represents a relative path to a
	// stategic merge patch with the format, url not supported
	PatchesStrategicMerge []PatchStrategicMerge `json:"patchesStrategicMerge,omitempty" yaml:"patchesStrategicMerge,omitempty"`

	PatchesJson6902 []Patch `json:"patchesJson6902,omitempty" yaml:"patchesJson6902,omitempty"`

	Patches []Patch `json:"patches,omitempty" yaml:"patches,omitempty"`

	// Resources specifies relative paths to files holding YAML representations
	// of kubernetes API objects, or specifications of other kustomizations
	// via relative paths, absolute paths, or URLs.
	Resources []string `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Components specifies relative paths to specifications of other Components
	// via relative paths, absolute paths, or URLs.
	Components []string `json:"components,omitempty" yaml:"components,omitempty"`

	// Crds specifies relative paths to Custom Resource Definition files.
	// This allows custom resources to be recognized as operands, making
	// it possible to add them to the Resources list.
	// CRDs themselves are not modified.
	Crds []string `json:"crds,omitempty" yaml:"crds,omitempty"`

	Bases []string `json:"bases,omitempty" yaml:"bases,omitempty"`

	// Configurations is a list of transformer configuration files
	Configurations []string `json:"configurations,omitempty" yaml:"configurations,omitempty"`

	// Generators is a list of files containing custom generators
	Generators []string `json:"generators,omitempty" yaml:"generators,omitempty"`

	// Transformers is a list of files containing transformers
	Transformers []string `json:"transformers,omitempty" yaml:"transformers,omitempty"`

	// Validators is a list of files containing validators
	Validators []string `json:"validators,omitempty" yaml:"validators,omitempty"`
}

type PatchStrategicMerge string

type Patch struct {
	// Path is a relative file path to the patch file.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}
