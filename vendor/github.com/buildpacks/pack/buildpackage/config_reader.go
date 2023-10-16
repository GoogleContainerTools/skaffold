package buildpackage

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

const defaultOS = "linux"

// Config encapsulates the possible configuration options for buildpackage creation.
type Config struct {
	Buildpack    dist.BuildpackURI `toml:"buildpack"`
	Extension    dist.BuildpackURI `toml:"extension"`
	Dependencies []dist.ImageOrURI `toml:"dependencies"`
	Platform     dist.Platform     `toml:"platform"`
}

func DefaultConfig() Config {
	return Config{
		Buildpack: dist.BuildpackURI{
			URI: ".",
		},
		Platform: dist.Platform{
			OS: defaultOS,
		},
	}
}

func DefaultExtensionConfig() Config {
	return Config{
		Extension: dist.BuildpackURI{
			URI: ".",
		},
		Platform: dist.Platform{
			OS: defaultOS,
		},
	}
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

	if packageConfig.Buildpack.URI == "" && packageConfig.Extension.URI == "" {
		if packageConfig.Buildpack.URI == "" {
			return packageConfig, errors.Errorf("missing %s configuration", style.Symbol("buildpack.uri"))
		}
		return packageConfig, errors.Errorf("missing %s configuration", style.Symbol("extension.uri"))
	}

	if packageConfig.Platform.OS == "" {
		packageConfig.Platform.OS = defaultOS
	}

	if packageConfig.Platform.OS != "linux" && packageConfig.Platform.OS != "windows" {
		return packageConfig, errors.Errorf("invalid %s configuration: only [%s, %s] is permitted, found %s",
			style.Symbol("platform.os"), style.Symbol("linux"), style.Symbol("windows"), style.Symbol(packageConfig.Platform.OS))
	}

	configDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return packageConfig, err
	}

	if err := validateURI(packageConfig.Buildpack.URI, configDir); err != nil {
		return packageConfig, err
	}

	for _, dep := range packageConfig.Dependencies {
		if dep.URI != "" && dep.ImageName != "" {
			return packageConfig, errors.Errorf(
				"dependency configured with both %s and %s",
				style.Symbol("uri"),
				style.Symbol("image"),
			)
		}

		if dep.URI != "" {
			if err := validateURI(dep.URI, configDir); err != nil {
				return packageConfig, err
			}
		}
	}

	return packageConfig, nil
}

func validateURI(uri, relativeBaseDir string) error {
	locatorType, err := buildpack.GetLocatorType(uri, relativeBaseDir, nil)
	if err != nil {
		return err
	}

	if locatorType == buildpack.InvalidLocator {
		return errors.Errorf("invalid locator %s", style.Symbol(uri))
	}

	return nil
}
