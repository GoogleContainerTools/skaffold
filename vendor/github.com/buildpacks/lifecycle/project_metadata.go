package lifecycle

const (
	ProjectMetadataLabel = "io.buildpacks.project.metadata"
)

type ProjectMetadata struct {
	Source *ProjectSource `toml:"source" json:"source,omitempty"`
}

type ProjectSource struct {
	Type     string                 `toml:"type" json:"type,omitempty"`
	Version  map[string]interface{} `toml:"version" json:"version,omitempty"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata,omitempty"`
}
