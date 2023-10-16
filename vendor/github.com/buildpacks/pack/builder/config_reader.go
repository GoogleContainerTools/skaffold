package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/dist"
)

// Config is a builder configuration file
type Config struct {
	Description     string           `toml:"description"`
	Buildpacks      ModuleCollection `toml:"buildpacks"`
	Extensions      ModuleCollection `toml:"extensions"`
	Order           dist.Order       `toml:"order"`
	OrderExtensions dist.Order       `toml:"order-extensions"`
	Stack           StackConfig      `toml:"stack"`
	Lifecycle       LifecycleConfig  `toml:"lifecycle"`
	Run             RunConfig        `toml:"run"`
	Build           BuildConfig      `toml:"build"`
}

// ModuleCollection is a list of ModuleConfigs
type ModuleCollection []ModuleConfig

// ModuleConfig details the configuration of a Buildpack or Extension
type ModuleConfig struct {
	dist.ModuleInfo
	dist.ImageOrURI
}

func (c *ModuleConfig) DisplayString() string {
	if c.ModuleInfo.FullName() != "" {
		return c.ModuleInfo.FullName()
	}

	return c.ImageOrURI.DisplayString()
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

// RunConfig set of run image configuration
type RunConfig struct {
	Images []RunImageConfig `toml:"images"`
}

// RunImageConfig run image id and mirrors
type RunImageConfig struct {
	Image   string   `toml:"image"`
	Mirrors []string `toml:"mirrors,omitempty"`
}

// BuildConfig build image configuration
type BuildConfig struct {
	Image string `toml:"image"`
}

// ReadConfig reads a builder configuration from the file path provided and returns the
// configuration along with any warnings encountered while parsing
func ReadConfig(path string) (config Config, warnings []string, err error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return Config{}, nil, errors.Wrap(err, "opening config file")
	}
	defer file.Close()

	config, err = parseConfig(file)
	if err != nil {
		return Config{}, nil, errors.Wrapf(err, "parse contents of '%s'", path)
	}

	if len(config.Order) == 0 {
		warnings = append(warnings, fmt.Sprintf("empty %s definition", style.Symbol("order")))
	}

	config.mergeStackWithImages()

	return config, warnings, nil
}

// ValidateConfig validates the config
func ValidateConfig(c Config) error {
	if c.Build.Image == "" && c.Stack.BuildImage == "" {
		return errors.New("build.image is required")
	} else if c.Build.Image != "" && c.Stack.BuildImage != "" && c.Build.Image != c.Stack.BuildImage {
		return errors.New("build.image and stack.build-image do not match")
	}

	if len(c.Run.Images) == 0 && (c.Stack.RunImage == "" || c.Stack.ID == "") {
		return errors.New("run.images are required")
	}

	for _, runImage := range c.Run.Images {
		if runImage.Image == "" {
			return errors.New("run.images.image is required")
		}
	}

	if c.Stack.RunImage != "" && c.Run.Images[0].Image != c.Stack.RunImage {
		return errors.New("run.images and stack.run-image do not match")
	}

	return nil
}

func (c *Config) mergeStackWithImages() {
	// RFC-0096
	if c.Build.Image != "" {
		c.Stack.BuildImage = c.Build.Image
	} else if c.Build.Image == "" && c.Stack.BuildImage != "" {
		c.Build.Image = c.Stack.BuildImage
	}

	if len(c.Run.Images) != 0 {
		// use the first run image as the "stack"
		c.Stack.RunImage = c.Run.Images[0].Image
		c.Stack.RunImageMirrors = c.Run.Images[0].Mirrors
	} else if len(c.Run.Images) == 0 && c.Stack.RunImage != "" {
		c.Run.Images = []RunImageConfig{{
			Image:   c.Stack.RunImage,
			Mirrors: c.Stack.RunImageMirrors,
		},
		}
	}
}

// parseConfig reads a builder configuration from file
func parseConfig(file *os.File) (Config, error) {
	builderConfig := Config{}
	tomlMetadata, err := toml.NewDecoder(file).Decode(&builderConfig)
	if err != nil {
		return Config{}, errors.Wrap(err, "decoding toml contents")
	}

	undecodedKeys := tomlMetadata.Undecoded()
	if len(undecodedKeys) > 0 {
		unknownElementsMsg := config.FormatUndecodedKeys(undecodedKeys)

		return Config{}, errors.Errorf("%s in %s",
			unknownElementsMsg,
			style.Symbol(file.Name()),
		)
	}

	return builderConfig, nil
}
