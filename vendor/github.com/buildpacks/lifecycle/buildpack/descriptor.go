package buildpack

const (
	KindBuildpack = "Buildpack"
	KindExtension = "Extension"
)

//go:generate mockgen -package testmock -destination ../testmock/component_descriptor.go github.com/buildpacks/lifecycle/buildpack Descriptor
type Descriptor interface {
	API() string
	Homepage() string
	TargetsList() []TargetMetadata
}

// BaseInfo is information shared by both buildpacks and extensions.
// For buildpacks it winds up under the toml `buildpack` key along with SBOM info, but extensions have no SBOMs.
type BaseInfo struct {
	ClearEnv bool   `toml:"clear-env,omitempty"`
	Homepage string `toml:"homepage,omitempty"`
	ID       string `toml:"id"`
	Name     string `toml:"name"`
	Version  string `toml:"version"`
}
