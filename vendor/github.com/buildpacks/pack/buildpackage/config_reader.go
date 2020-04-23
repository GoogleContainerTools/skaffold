package buildpackage

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
)

// Config encapsulates the possible configuration options for buildpackage creation.
type Config struct {
	Buildpack    dist.BuildpackURI `toml:"buildpack"`
	Dependencies []dist.ImageOrURI `toml:"dependencies"`
}

// NewConfigReader returns an instance of ConfigReader. It does not take any parameters.
func NewConfigReader() *ConfigReader {
	return &ConfigReader{}
}

// ConfigReader implements a Read method for buildpackage configuration which parses and validates buildpackage
// configuration from a toml file.
type ConfigReader struct{}

// Read reads and validates a buildpackage configuration from the file path provided and returns the
// configuration and any error that occurred during reading or validation.
func (r *ConfigReader) Read(path string) (Config, error) {
	packageConfig := Config{}

	tomlMetadata, err := toml.DecodeFile(path, &packageConfig)
	if err != nil {
		return packageConfig, errors.Wrap(err, "decoding toml")
	}

	undecodedKeys := tomlMetadata.Undecoded()
	if len(undecodedKeys) > 0 {
		unknownElementsMsg := config.FormatUndecodedKeys(undecodedKeys)

		return packageConfig, errors.Errorf("%s in %s",
			unknownElementsMsg,
			style.Symbol(path),
		)
	}

	if packageConfig.Buildpack.URI == "" {
		return packageConfig, errors.Errorf("missing %s configuration", style.Symbol("buildpack.uri"))
	}

	configDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return packageConfig, err
	}

	absPath, err := paths.ToAbsolute(packageConfig.Buildpack.URI, configDir)
	if err != nil {
		return packageConfig, errors.Wrapf(err, "getting absolute path for %s", style.Symbol(packageConfig.Buildpack.URI))
	}
	packageConfig.Buildpack.URI = absPath

	for i := range packageConfig.Dependencies {
		uri := packageConfig.Dependencies[i].URI
		if uri != "" {
			absPath, err := paths.ToAbsolute(uri, configDir)
			if err != nil {
				return packageConfig, errors.Wrapf(err, "getting absolute path for %s", style.Symbol(uri))
			}

			packageConfig.Dependencies[i].URI = absPath
		}

		dep := packageConfig.Dependencies[i]
		if dep.URI != "" && dep.ImageName != "" {
			return packageConfig, errors.Errorf(
				"dependency configured with both %s and %s",
				style.Symbol("uri"),
				style.Symbol("image"),
			)
		}
	}

	return packageConfig, nil
}
