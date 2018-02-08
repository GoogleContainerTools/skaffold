package cli

// Default build-time variable.
// These values are overriding via ldflags
var (
	PlatformName = ""
	Version      = "unknown-version"
	GitCommit    = "unknown-commit"
	BuildTime    = "unknown-buildtime"
)
