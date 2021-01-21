/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func getAllConfigs(opts config.SkaffoldOptions) ([]*latest.SkaffoldConfig, error) {
	cfgs, err := getConfigs(opts.ConfigurationFile, nil, opts.Profiles, opts, make(map[string]string), false, false, make(map[string]string))
	if err != nil {
		return nil, err
	}
	if len(cfgs) == 0 {
		if len(opts.ConfigurationFilter) > 0 {
			return nil, fmt.Errorf("did not find any configs matching selection %v", opts.ConfigurationFilter)
		}
		return nil, fmt.Errorf("failed to get any valid configs from %s", opts.ConfigurationFile)
	}
	return cfgs, nil
}

func getConfigs(configFile string, configSelection []string, profileSelection []string, opts config.SkaffoldOptions, appliedProfiles map[string]string, requiredConfigs bool, isDependencyConfig bool, configNameToFile map[string]string) ([]*latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(configFile, latest.Version)
	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(configFile) {
		cwd, _ := util.RealWorkDir()
		// convert `configFile` to absolute value as it's used as a map key in several places.
		configFile = filepath.Join(cwd, configFile)
	}

	if len(parsed) == 0 {
		return nil, fmt.Errorf("skaffold config file %s is empty", opts.ConfigurationFile)
	}

	var configs []*latest.SkaffoldConfig
	for i, cfg := range parsed {
		config := cfg.(*latest.SkaffoldConfig)

		// check that the same config name isn't repeated in multiple files.
		if config.Metadata.Name != "" {
			prevConfig, found := configNameToFile[config.Metadata.Name]
			if found && prevConfig != configFile {
				return nil, fmt.Errorf("skaffold config named %q found in multiple files: %q and %q", config.Metadata.Name, prevConfig, configFile)
			}
			configNameToFile[config.Metadata.Name] = configFile
		}

		// configSelection specifies the exact required configs in this file. Empty configSelection means that all configs are required.
		if len(configSelection) > 0 && !util.StrSliceContains(configSelection, config.Metadata.Name) {
			continue
		}

		// if config names are explicitly specified via the configuration flag, then need to include the dependency tree of configs starting at that named config.
		// `requiredConfigs` specifies if we are already in the dependency-tree of a required config, so all selected configs are required even if they are not explicitly named va the configuration flag.
		required := requiredConfigs || len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, config.Metadata.Name)

		profiles, err := schema.ApplyProfiles(config, opts, profileSelection)
		if err != nil {
			return nil, fmt.Errorf("applying profiles: %w", err)
		}
		if err := defaults.Set(config); err != nil {
			return nil, fmt.Errorf("setting default values: %w", err)
		}
		// convert relative file paths to absolute for all configs that are not in invoked explicitly. This avoids maintaining multiple root directory information since the dependency skaffold configs would have their own root directory.
		if isDependencyConfig {
			if err := tags.MakeFilePathsAbsolute(config, filepath.Dir(configFile)); err != nil {
				return nil, fmt.Errorf("setting absolute filepaths: %w", err)
			}
		}

		sort.Strings(profiles)
		key := fmt.Sprintf("%s:%d:%t", configFile, i, required)
		expected := strings.Join(profiles, ",")
		// check that this config was not previously referenced with a different set of active profiles.
		// including `required` in the key implies that we search this dependency tree once for a possible match of required named configs, but also again if the current config is itself required which makes all subsequent configs also required.
		if previous, found := appliedProfiles[key]; found {
			if previous != expected {
				configID := fmt.Sprintf("index %d", i)
				if config.Metadata.Name != "" {
					configID = config.Metadata.Name
				}
				return nil, fmt.Errorf("skaffold config %s from file %s imported multiple times with different profiles", configID, configFile)
			}
			continue
		}
		appliedProfiles[key] = expected

		for _, d := range config.Dependencies {
			var depProfiles []string
			for _, ap := range d.ActiveProfiles {
				if len(ap.ActivatedBy) == 0 {
					depProfiles = append(depProfiles, ap.Name)
					continue
				}
				for _, p := range profiles {
					if util.StrSliceContains(ap.ActivatedBy, p) {
						depProfiles = append(depProfiles, ap.Name)
						break
					}
				}
			}
			path := d.Path
			if path == "" {
				// empty path means configs in the same file
				path = configFile
			}
			fi, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(errors.Unwrap(err)) {
					return nil, fmt.Errorf("could not find skaffold config %s that is referenced as a dependency in config %s", d.Path, configFile)
				}
				return nil, fmt.Errorf("parsing dependencies for skaffold config %s: %w", configFile, err)
			}
			if fi.IsDir() {
				path = filepath.Join(path, "skaffold.yaml")
			}
			depConfigs, err := getConfigs(path, d.Names, depProfiles, opts, appliedProfiles, required, path != configFile, configNameToFile)
			if err != nil {
				return nil, err
			}
			configs = append(configs, depConfigs...)
		}

		if required {
			configs = append(configs, config)
		}
	}
	return configs, nil
}
