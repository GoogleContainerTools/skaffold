package files

// system.toml is provided by the platform or builder to define system buildpacks
// that should be prepended or appended to the order during detection.
// See `buildpack.Order` for further information.

// System represents the contents of the system.toml file.
type System struct {
	Pre  SystemBuildpacks `toml:"pre"`
	Post SystemBuildpacks `toml:"post"`
}

// SystemBuildpacks contains the list of buildpacks to include.
type SystemBuildpacks struct {
	Buildpacks []SystemBuildpack `toml:"buildpacks"`
}

// SystemBuildpack represents a buildpack reference in the system.toml file.
type SystemBuildpack struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
}
