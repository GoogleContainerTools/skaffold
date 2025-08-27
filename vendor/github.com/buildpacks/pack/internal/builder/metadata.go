package builder

import "github.com/buildpacks/pack/pkg/dist"

const (
	OrderLabel           = "io.buildpacks.buildpack.order"
	OrderExtensionsLabel = "io.buildpacks.buildpack.order-extensions"
)

type Metadata struct {
	Description string             `json:"description"`
	Buildpacks  []dist.ModuleInfo  `json:"buildpacks"`
	Extensions  []dist.ModuleInfo  `json:"extensions"`
	Stack       StackMetadata      `json:"stack"`
	Lifecycle   LifecycleMetadata  `json:"lifecycle"`
	CreatedBy   CreatorMetadata    `json:"createdBy"`
	RunImages   []RunImageMetadata `json:"images"`
}

type CreatorMetadata struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type LifecycleMetadata struct {
	LifecycleInfo
	// Deprecated: use APIs instead
	API  LifecycleAPI  `json:"api"`
	APIs LifecycleAPIs `json:"apis"`
}

type StackMetadata struct {
	RunImage RunImageMetadata `json:"runImage" toml:"run-image"`
}

type RunImages struct {
	Images []RunImageMetadata `json:"images" toml:"images"`
}

type RunImageMetadata struct {
	Image   string   `json:"image" toml:"image"`
	Mirrors []string `json:"mirrors" toml:"mirrors"`
}
