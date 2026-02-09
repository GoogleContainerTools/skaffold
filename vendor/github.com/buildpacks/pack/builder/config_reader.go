package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Targets         []dist.Target    `toml:"targets"`
	System          dist.System      `toml:"system"`
}

// ModuleCollection is a list of ModuleConfigs
type ModuleCollection []ModuleConfig

// ModuleConfig details the configuration of a Buildpack or Extension
type ModuleConfig struct {
	dist.ModuleInfo
	dist.ImageOrURI
}

func (c *ModuleConfig) DisplayString() string {
	if c.FullName() != "" {
		return c.FullName()
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
	Image string           `toml:"image"`
	Env   []BuildConfigEnv `toml:"env"`
}

type Suffix string

const (
	NONE     Suffix = ""
	DEFAULT  Suffix = "default"
	OVERRIDE Suffix = "override"
	APPEND   Suffix = "append"
	PREPEND  Suffix = "prepend"
)

type BuildConfigEnv struct {
	Name    string   `toml:"name"`
	Value   string   `toml:"value"`
	Suffix  Suffix   `toml:"suffix,omitempty"`
	Delim   string   `toml:"delim,omitempty"`
	ExecEnv []string `toml:"exec-env,omitempty"`
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

func ParseBuildConfigEnv(env []BuildConfigEnv, path string) (envMap map[string]string, warnings []string, err error) {
	envMap = map[string]string{}
	var appendOrPrependWithoutDelim = 0
	for _, v := range env {
		if name := v.Name; name == "" || len(name) == 0 {
			return nil, nil, errors.Wrapf(errors.Errorf("env name should not be empty"), "parse contents of '%s'", path)
		}
		if val := v.Value; val == "" || len(val) == 0 {
			warnings = append(warnings, fmt.Sprintf("empty value for key/name %s", style.Symbol(v.Name)))
		}
		suffixName, delimName, err := getBuildConfigEnvFileName(v)
		if err != nil {
			return envMap, warnings, err
		}
		if val, e := envMap[suffixName]; e {
			warnings = append(warnings, fmt.Sprintf(errors.Errorf("overriding env with name: %s and suffix: %s from %s to %s", style.Symbol(v.Name), style.Symbol(string(v.Suffix)), style.Symbol(val), style.Symbol(v.Value)).Error(), "parse contents of '%s'", path))
		}
		if val, e := envMap[delimName]; e {
			warnings = append(warnings, fmt.Sprintf(errors.Errorf("overriding env with name: %s and delim: %s from %s to %s", style.Symbol(v.Name), style.Symbol(v.Delim), style.Symbol(val), style.Symbol(v.Value)).Error(), "parse contents of '%s'", path))
		}
		if delim := v.Delim; (delim != "" || len(delim) != 0) && (delimName != "" || len(delimName) != 0) {
			envMap[delimName] = delim
		}
		envMap[suffixName] = v.Value
	}

	for k := range envMap {
		name, suffix, err := getFilePrefixSuffix(k)
		if err != nil {
			continue
		}
		if _, ok := envMap[name+".delim"]; (suffix == "append" || suffix == "prepend") && !ok {
			warnings = append(warnings, fmt.Sprintf(errors.Errorf("env with name/key %s with suffix %s must to have a %s value", style.Symbol(name), style.Symbol(suffix), style.Symbol("delim")).Error(), "parse contents of '%s'", path))
			appendOrPrependWithoutDelim++
		}
	}
	if appendOrPrependWithoutDelim > 0 {
		return envMap, warnings, errors.Errorf("error parsing [[build.env]] in file '%s'", path)
	}
	return envMap, warnings, err
}

func getBuildConfigEnvFileName(env BuildConfigEnv) (suffixName, delimName string, err error) {
	suffix, err := getActionType(env.Suffix)
	if err != nil {
		return suffixName, delimName, err
	}
	if suffix == "" {
		suffixName = env.Name
	} else {
		suffixName = env.Name + suffix
	}
	if delim := env.Delim; delim != "" || len(delim) != 0 {
		delimName = env.Name + ".delim"
	}
	return suffixName, delimName, err
}

func getActionType(suffix Suffix) (suffixString string, err error) {
	const delim = "."
	switch suffix {
	case NONE:
		return "", nil
	case DEFAULT:
		return delim + string(DEFAULT), nil
	case OVERRIDE:
		return delim + string(OVERRIDE), nil
	case APPEND:
		return delim + string(APPEND), nil
	case PREPEND:
		return delim + string(PREPEND), nil
	default:
		return suffixString, errors.Errorf("unknown action type %s", style.Symbol(string(suffix)))
	}
}

func getFilePrefixSuffix(filename string) (prefix, suffix string, err error) {
	val := strings.Split(filename, ".")
	if len(val) <= 1 {
		return val[0], suffix, errors.Errorf("Suffix might be null")
	}
	if len(val) == 2 {
		suffix = val[1]
	} else {
		suffix = strings.Join(val[1:], ".")
	}
	return val[0], suffix, err
}
