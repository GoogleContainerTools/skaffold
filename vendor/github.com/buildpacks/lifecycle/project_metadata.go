package lifecycle

const (
	ProjectMetadataLabel = "io.buildpacks.project.metadata"
)

type ProjectMetadata struct {
	Source ProjectSource `toml:"source" json:"source"`
}

type ProjectSource struct {
	Type     string                 `toml:"type" json:"type"`
	Version  map[string]interface{} `toml:"version" json:"version"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata"`
}
