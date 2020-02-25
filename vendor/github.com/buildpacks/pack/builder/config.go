package builder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
)

type Config struct {
	Description string            `toml:"description"`
	Buildpacks  []BuildpackConfig `toml:"buildpacks"`
	Packages    []PackageConfig   `toml:"packages"`
	Order       dist.Order        `toml:"order"`
	Stack       StackConfig       `toml:"stack"`
	Lifecycle   LifecycleConfig   `toml:"lifecycle"`
}

type BuildpackConfig struct {
	dist.BuildpackInfo
	URI string `toml:"uri"`
}

type PackageConfig struct {
	Image string `toml:"image"`
}

type StackConfig struct {
	ID              string   `toml:"id"`
	BuildImage      string   `toml:"build-image"`
	RunImage        string   `toml:"run-image"`
	RunImageMirrors []string `toml:"run-image-mirrors,omitempty"`
}

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

	warnings, err = getWarningsForObsoleteFields(file)
	if err != nil {
		return Config{}, nil, errors.Wrapf(err, "check warnings for file '%s'", path)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Config{}, nil, errors.Wrap(err, "reset config file pointer")
	}

	config, err = parseConfig(file, builderDir)
	if err != nil {
		return Config{}, nil, errors.Wrapf(err, "parse contents of '%s'", path)
	}

	if len(config.Order) == 0 {
		warnings = append(warnings, fmt.Sprintf("empty %s definition", style.Symbol("order")))
	}

	return config, warnings, nil
}

func getWarningsForObsoleteFields(reader io.Reader) ([]string, error) {
	var warnings []string

	var obsoleteConfig = struct {
		Buildpacks []struct {
			Latest bool
		}
		Groups []interface{}
	}{}

	if _, err := toml.DecodeReader(reader, &obsoleteConfig); err != nil {
		return nil, err
	}

	latestUsed := false

	for _, bp := range obsoleteConfig.Buildpacks {
		latestUsed = bp.Latest
	}

	if latestUsed {
		warnings = append(warnings, fmt.Sprintf("%s field on a buildpack is obsolete and will be ignored", style.Symbol("latest")))
	}

	if len(obsoleteConfig.Groups) > 0 {
		warnings = append(warnings, fmt.Sprintf("%s field is obsolete in favor of %s", style.Symbol("groups"), style.Symbol("order")))
	}

	return warnings, nil
}

// parseConfig reads a builder configuration from reader and resolves relative buildpack paths using `relativeToDir`
func parseConfig(reader io.Reader, relativeToDir string) (Config, error) {
	var builderConfig Config
	if _, err := toml.DecodeReader(reader, &builderConfig); err != nil {
		return Config{}, errors.Wrap(err, "decoding toml contents")
	}

	for i, bp := range builderConfig.Buildpacks {
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
