package files

// ProjectMetadata is written as `project_metadata.toml` by the exporter to record information about the application source code.
// It is also serialized by the exporter as the `io.buildpacks.project.metadata` label on the output image.
// The location of the file can be specified by providing `-project-metadata <path>` to the lifecycle.
type ProjectMetadata struct {
	Source *ProjectSource `toml:"source" json:"source,omitempty"`
}

type ProjectSource struct {
	Type     string                 `toml:"type" json:"type,omitempty"`
	Version  map[string]interface{} `toml:"version" json:"version,omitempty"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata,omitempty"`
}
