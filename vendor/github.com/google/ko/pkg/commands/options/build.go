// Copyright 2019 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/tools/go/packages"

	"github.com/google/ko/pkg/build"
)

const (
	// configDefaultBaseImage is the default base image if not specified in .ko.yaml.
	configDefaultBaseImage = "cgr.dev/chainguard/static:latest"
)

// BuildOptions represents options for the ko builder.
type BuildOptions struct {
	// BaseImage enables setting the default base image programmatically.
	// If non-empty, this takes precedence over the value in `.ko.yaml`.
	BaseImage string

	// BaseImageOverrides stores base image overrides for import paths.
	BaseImageOverrides map[string]string

	// DefaultPlatforms defines the default platforms when Platforms is not explicitly defined
	DefaultPlatforms []string

	// WorkingDirectory allows for setting the working directory for invocations of the `go` tool.
	// Empty string means the current working directory.
	WorkingDirectory string

	ConcurrentBuilds     int
	DisableOptimizations bool
	SBOM                 string
	SBOMDir              string
	Platforms            []string
	Labels               []string
	// UserAgent enables overriding the default value of the `User-Agent` HTTP
	// request header used when retrieving the base image.
	UserAgent string

	InsecureRegistry bool

	// Trimpath controls whether ko adds the `-trimpath` flag to `go build` by default.
	// The `-trimpath` flags aids in achieving reproducible builds, but it removes path information that is useful for interactive debugging.
	// Set this field to `false` and `DisableOptimizations` to `true` if you want to interactively debug the binary in the resulting image.
	// `AddBuildOptions()` defaults this field to `true`.
	Trimpath bool

	// BuildConfigs stores the per-image build config from `.ko.yaml`.
	BuildConfigs map[string]build.Config
}

func AddBuildOptions(cmd *cobra.Command, bo *BuildOptions) {
	cmd.Flags().IntVarP(&bo.ConcurrentBuilds, "jobs", "j", 0,
		"The maximum number of concurrent builds (default GOMAXPROCS)")
	cmd.Flags().BoolVar(&bo.DisableOptimizations, "disable-optimizations", bo.DisableOptimizations,
		"Disable optimizations when building Go code. Useful when you want to interactively debug the created container.")
	cmd.Flags().StringVar(&bo.SBOM, "sbom", "spdx",
		"The SBOM media type to use (none will disable SBOM synthesis and upload, also supports: spdx, cyclonedx, go.version-m).")
	cmd.Flags().StringVar(&bo.SBOMDir, "sbom-dir", "",
		"Path to file where the SBOM will be written.")
	cmd.Flags().StringSliceVar(&bo.Platforms, "platform", []string{},
		"Which platform to use when pulling a multi-platform base. Format: all | <os>[/<arch>[/<variant>]][,platform]*")
	cmd.Flags().StringSliceVar(&bo.Labels, "image-label", []string{},
		"Which labels (key=value) to add to the image.")
	bo.Trimpath = true
}

// LoadConfig reads build configuration from defaults, environment variables, and the `.ko.yaml` config file.
func (bo *BuildOptions) LoadConfig() error {
	v := viper.New()
	if bo.WorkingDirectory == "" {
		bo.WorkingDirectory = "."
	}
	// If omitted, use this base image.
	v.SetDefault("defaultBaseImage", configDefaultBaseImage)
	const configName = ".ko"

	v.SetConfigName(configName) // .yaml is implicit
	v.SetEnvPrefix("KO")
	v.AutomaticEnv()

	if override := os.Getenv("KO_CONFIG_PATH"); override != "" {
		file, err := os.Stat(override)
		if err != nil {
			return fmt.Errorf("error looking for config file: %w", err)
		}
		if file.Mode().IsRegular() {
			v.SetConfigFile(override)
		} else if file.IsDir() {
			path := filepath.Join(override, ".ko.yaml")
			file, err = os.Stat(path)
			if err != nil {
				return fmt.Errorf("error looking for config file: %w", err)
			}
			if file.Mode().IsRegular() {
				v.SetConfigFile(path)
			} else {
				return fmt.Errorf("config file %s is not a regular file", path)
			}
		} else {
			return fmt.Errorf("config file %s is not a regular file", override)
		}
	}
	v.AddConfigPath(bo.WorkingDirectory)

	if err := v.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	dp := v.GetStringSlice("defaultPlatforms")
	if len(dp) > 0 {
		bo.DefaultPlatforms = dp
	}

	if bo.BaseImage == "" {
		ref := v.GetString("defaultBaseImage")
		if _, err := name.ParseReference(ref); err != nil {
			return fmt.Errorf("'defaultBaseImage': error parsing %q as image reference: %w", ref, err)
		}
		bo.BaseImage = ref
	}

	if len(bo.BaseImageOverrides) == 0 {
		baseImageOverrides := map[string]string{}
		overrides := v.GetStringMapString("baseImageOverrides")
		for key, value := range overrides {
			if _, err := name.ParseReference(value); err != nil {
				return fmt.Errorf("'baseImageOverrides': error parsing %q as image reference: %w", value, err)
			}
			baseImageOverrides[key] = value
		}
		bo.BaseImageOverrides = baseImageOverrides
	}

	if len(bo.BuildConfigs) == 0 {
		var builds []build.Config
		if err := v.UnmarshalKey("builds", &builds); err != nil {
			return fmt.Errorf("configuration section 'builds' cannot be parsed")
		}
		buildConfigs, err := createBuildConfigMap(bo.WorkingDirectory, builds)
		if err != nil {
			return fmt.Errorf("could not create build config map: %w", err)
		}
		bo.BuildConfigs = buildConfigs
	}

	return nil
}

func createBuildConfigMap(workingDirectory string, configs []build.Config) (map[string]build.Config, error) {
	buildConfigsByImportPath := make(map[string]build.Config)
	for i, config := range configs {
		// In case no ID is specified, use the index of the build config in
		// the ko YAML file as a reference (debug help).
		if config.ID == "" {
			config.ID = fmt.Sprintf("#%d", i)
		}

		// Make sure to behave like GoReleaser by defaulting to the current
		// directory in case the build or main field is not set, check
		// https://goreleaser.com/customization/build/ for details
		if config.Dir == "" {
			config.Dir = "."
		}
		if config.Main == "" {
			config.Main = "."
		}

		// baseDir is the directory where `go list` will be run to look for package information
		baseDir := filepath.Join(workingDirectory, config.Dir)

		// To behave like GoReleaser, check whether the configured `main` config value points to a
		// source file, and if so, just use the directory it is in
		path := config.Main
		if fi, err := os.Stat(filepath.Join(baseDir, config.Main)); err == nil && fi.Mode().IsRegular() {
			path = filepath.Dir(config.Main)
		}

		// Verify that the path actually leads to a local file (https://github.com/google/ko/issues/483)
		if _, err := os.Stat(filepath.Join(baseDir, path)); err != nil {
			return nil, err
		}

		// By default, paths configured in the builds section are considered
		// local import paths, therefore add a "./" equivalent as a prefix to
		// the constructured import path
		localImportPath := fmt.Sprint(".", string(filepath.Separator), path)
		dir := filepath.Clean(baseDir)
		if dir == "." {
			dir = ""
		}
		pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName, Dir: dir}, localImportPath)
		if err != nil {
			return nil, fmt.Errorf("'builds': entry #%d does not contain a valid local import path (%s) for directory (%s): %w", i, localImportPath, baseDir, err)
		}

		if len(pkgs) != 1 {
			return nil, fmt.Errorf("'builds': entry #%d results in %d local packages, only 1 is expected", i, len(pkgs))
		}
		importPath := pkgs[0].PkgPath
		buildConfigsByImportPath[importPath] = config
	}

	return buildConfigsByImportPath, nil
}
