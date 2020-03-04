package buildpackage

import (
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

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
	config := Config{}

	tomlMetadata, err := toml.DecodeFile(path, &config)
	if err != nil {
		return config, errors.Wrap(err, "decoding toml")
	}

	undecodedKeys := tomlMetadata.Undecoded()
	if len(undecodedKeys) > 0 {
		unusedKeys := map[string]interface{}{}
		for _, key := range undecodedKeys {
			keyName := key.String()

			parent := strings.Split(keyName, ".")[0]

			if _, ok := unusedKeys[parent]; !ok {
				unusedKeys[keyName] = nil
			}
		}

		var errorKeys []string
		for errorKey := range unusedKeys {
			errorKeys = append(errorKeys, style.Symbol(errorKey))
		}

		pluralizedElement := "element"
		if len(errorKeys) > 1 {
			pluralizedElement += "s"
		}

		return config, errors.Errorf("unknown configuration %s %s in %s",
			pluralizedElement,
			strings.Join(errorKeys, ", "),
			style.Symbol(path),
		)
	}

	if config.Buildpack.URI == "" {
		return config, errors.Errorf("missing %s configuration", style.Symbol("buildpack.uri"))
	}

	configDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return config, err
	}

	absPath, err := paths.ToAbsolute(config.Buildpack.URI, configDir)
	if err != nil {
		return config, errors.Wrapf(err, "getting absolute path for %s", style.Symbol(config.Buildpack.URI))
	}
	config.Buildpack.URI = absPath

	for i := range config.Dependencies {
		uri := config.Dependencies[i].URI
		if uri != "" {
			absPath, err := paths.ToAbsolute(uri, configDir)
			if err != nil {
				return config, errors.Wrapf(err, "getting absolute path for %s", style.Symbol(uri))
			}

			config.Dependencies[i].URI = absPath
		}

		dep := config.Dependencies[i]
		if dep.URI != "" && dep.ImageName != "" {
			return config, errors.Errorf(
				"dependency configured with both %s and %s",
				style.Symbol("uri"),
				style.Symbol("image"),
			)
		}
	}

	return config, nil
}
