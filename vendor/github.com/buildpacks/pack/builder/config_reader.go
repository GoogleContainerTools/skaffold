package builder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
)

// Config is a builder configuration file
type Config struct {
	Description string              `toml:"description"`
	Buildpacks  BuildpackCollection `toml:"buildpacks"`
	Order       dist.Order          `toml:"order"`
	Stack       StackConfig         `toml:"stack"`
	Lifecycle   LifecycleConfig     `toml:"lifecycle"`
}

// BuildpackCollection is a list of BuildpackConfigs
type BuildpackCollection []BuildpackConfig

// BuildpackConfig details the configuration of a Buildpack
type BuildpackConfig struct {
	dist.BuildpackInfo
	dist.ImageOrURI
}

// StackConfig details the configuration of a Stack
type StackConfig struct {
	ID              string   `toml:"id"`
	BuildImage      string   `toml:"build-image"`
	RunImage        string   `toml:"run-image"`
	RunImageMirrors []string `toml:"run-image-mirrors,omitempty"`
}

// LifecycleConfig details the configuration of the Lifecycle
type LifecycleConfig struct {
	URI     string `toml:"uri"`
	Version string `toml:"version"`
}

// ReadConfig reads a builder configuration from the file path provided and returns the
// configuration along with any warnings encountered while parsing
func ReadConfig(path string) (config Config, warnings []string, err error) {
	builderDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return Config{}, nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return Config{}, nil, errors.Wrap(err, "opening config file")
	}
	defer file.Close()

	config, err = parseConfig(file, builderDir, path)
	if err != nil {
		return Config{}, nil, errors.Wrapf(err, "parse contents of '%s'", path)
	}

	if len(config.Order) == 0 {
		warnings = append(warnings, fmt.Sprintf("empty %s definition", style.Symbol("order")))
	}

	return config, warnings, nil
}

// ValidateConfig validates the config
func ValidateConfig(c Config) error {
	if c.Stack.ID == "" {
		return errors.New("stack.id is required")
	}

	if c.Stack.BuildImage == "" {
		return errors.New("stack.build-image is required")
	}

	if c.Stack.RunImage == "" {
		return errors.New("stack.run-image is required")
	}

	return nil
}

// parseConfig reads a builder configuration from reader and resolves relative buildpack paths using `relativeToDir`
func parseConfig(reader io.Reader, relativeToDir, path string) (Config, error) {
	builderConfig := Config{}
	tomlMetadata, err := toml.DecodeReader(reader, &builderConfig)
	if err != nil {
		return Config{}, errors.Wrap(err, "decoding toml contents")
	}

	undecodedKeys := tomlMetadata.Undecoded()
	if len(undecodedKeys) > 0 {
		unknownElementsMsg := config.FormatUndecodedKeys(undecodedKeys)

		return Config{}, errors.Errorf("%s in %s",
			unknownElementsMsg,
			style.Symbol(path),
		)
	}

	for i, bp := range builderConfig.Buildpacks {
		if bp.URI == "" {
			continue
		}

		uri, err := paths.ToAbsolute(bp.URI, relativeToDir)
		if err != nil {
			return Config{}, errors.Wrap(err, "transforming buildpack URI")
		}
		builderConfig.Buildpacks[i].URI = uri
	}

	if builderConfig.Lifecycle.URI != "" {
		uri, err := paths.ToAbsolute(builderConfig.Lifecycle.URI, relativeToDir)
		if err != nil {
			return Config{}, errors.Wrap(err, "transforming lifecycle URI")
		}
		builderConfig.Lifecycle.URI = uri
	}

	return builderConfig, nil
}
