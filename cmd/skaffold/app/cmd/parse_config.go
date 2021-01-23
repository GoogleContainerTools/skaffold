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

type configOpts struct {
	// path to the `skaffold.yaml` file
	file string
	// names of configs to select from this file.
	selection []string
	// list of profiles to apply to the selection
	profiles []string
	// is this a required config.
	isRequired bool
	// is this config resolved as a dependency as opposed to being set explicitly (via the `-f` flag)
	isDependency bool
}

func getAllConfigs(opts config.SkaffoldOptions) ([]*latest.SkaffoldConfig, error) {
	cOpts := configOpts{file: opts.ConfigurationFile, selection: nil, profiles: opts.Profiles, isRequired: false, isDependency: false}
	cfgs, err := getConfigs(cOpts, opts, make(map[string]string), make(map[string]string))
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

// getConfigs recursively parses all configs and their dependencies in the specified `skaffold.yaml`
func getConfigs(cfgOpts configOpts, opts config.SkaffoldOptions, appliedProfiles map[string]string, configNameToFile map[string]string) ([]*latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(cfgOpts.file, latest.Version)
	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(cfgOpts.file) {
		cwd, _ := util.RealWorkDir()
		// convert `file` path to absolute value as it's used as a map key in several places.
		cfgOpts.file = filepath.Join(cwd, cfgOpts.file)
	}

	if len(parsed) == 0 {
		return nil, fmt.Errorf("skaffold config file %s is empty", opts.ConfigurationFile)
	}

	var configs []*latest.SkaffoldConfig
	for i, cfg := range parsed {
		config := cfg.(*latest.SkaffoldConfig)
		processed, err := processEachConfig(config, cfgOpts, opts, appliedProfiles, configNameToFile, i)
		if err != nil {
			return nil, err
		}
		configs = append(configs, processed...)
	}
	return configs, nil
}

// processEachConfig processes each parsed config by applying profiles and recursively processing it's dependencies
func processEachConfig(config *latest.SkaffoldConfig, cfgOpts configOpts, opts config.SkaffoldOptions, appliedProfiles map[string]string, configNameToFile map[string]string, index int) ([]*latest.SkaffoldConfig, error) {
	// check that the same config name isn't repeated in multiple files.
	if config.Metadata.Name != "" {
		prevConfig, found := configNameToFile[config.Metadata.Name]
		if found && prevConfig != cfgOpts.file {
			return nil, fmt.Errorf("skaffold config named %q found in multiple files: %q and %q", config.Metadata.Name, prevConfig, cfgOpts.file)
		}
		configNameToFile[config.Metadata.Name] = cfgOpts.file
	}

	// configSelection specifies the exact required configs in this file. Empty configSelection means that all configs are required.
	if len(cfgOpts.selection) > 0 && !util.StrSliceContains(cfgOpts.selection, config.Metadata.Name) {
		return nil, nil
	}

	// if config names are explicitly specified via the configuration flag, we need to include the dependency tree of configs starting at that named config.
	// `requiredConfigs` specifies if we are already in the dependency-tree of a required config, so all selected configs are required even if they are not explicitly named via the configuration flag.
	required := cfgOpts.isRequired || len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, config.Metadata.Name)

	profiles, err := schema.ApplyProfiles(config, opts, cfgOpts.profiles)
	if err != nil {
		return nil, fmt.Errorf("applying profiles: %w", err)
	}
	if err := defaults.Set(config); err != nil {
		return nil, fmt.Errorf("setting default values: %w", err)
	}
	// convert relative file paths to absolute for all configs that are not invoked explicitly. This avoids maintaining multiple root directory information since the dependency skaffold configs would have their own root directory.
	if cfgOpts.isDependency {
		if err := tags.MakeFilePathsAbsolute(config, filepath.Dir(cfgOpts.file)); err != nil {
			return nil, fmt.Errorf("setting absolute filepaths: %w", err)
		}
	}

	sort.Strings(profiles)
	if revisit, err := checkRevisit(config, profiles, appliedProfiles, cfgOpts.file, required, index); revisit {
		return nil, err
	}

	var configs []*latest.SkaffoldConfig
	for _, d := range config.Dependencies {
		depConfigs, err := processEachDependency(d, cfgOpts.file, required, profiles, opts, appliedProfiles, configNameToFile)
		if err != nil {
			return nil, err
		}
		configs = append(configs, depConfigs...)
	}

	if required {
		configs = append(configs, config)
	}
	return configs, nil
}

// processEachDependency parses a config dependency with the calculated set of activated profiles.
func processEachDependency(d latest.ConfigDependency, fPath string, required bool, profiles []string, opts config.SkaffoldOptions, appliedProfiles, configNameToFile map[string]string) ([]*latest.SkaffoldConfig, error) {
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
		path = fPath
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return nil, fmt.Errorf("could not find skaffold config %s that is referenced as a dependency in config %s", d.Path, fPath)
		}
		return nil, fmt.Errorf("parsing dependencies for skaffold config %s: %w", fPath, err)
	}
	if fi.IsDir() {
		path = filepath.Join(path, "skaffold.yaml")
	}
	depConfigs, err := getConfigs(configOpts{file: path, selection: d.Names, profiles: depProfiles, isRequired: required, isDependency: path != fPath}, opts, appliedProfiles, configNameToFile)
	if err != nil {
		return nil, err
	}
	return depConfigs, nil
}

// checkRevisit ensures that each config is activated with the same set of active profiles
// It returns true if this config was visited once before. It additionally returns an error if the previous visit was with a different set of active profiles.
func checkRevisit(config *latest.SkaffoldConfig, profiles []string, appliedProfiles map[string]string, file string, required bool, index int) (bool, error) {
	key := fmt.Sprintf("%s:%d:%t", file, index, required)
	expected := strings.Join(profiles, ",")
	// including `required` in the key implies that we search this dependency tree once for a possible match of required named configs, but also again if the current config is itself required which makes all subsequent configs also required.
	if previous, found := appliedProfiles[key]; found {
		if previous != expected {
			configID := fmt.Sprintf("index %d", index)
			if config.Metadata.Name != "" {
				configID = config.Metadata.Name
			}
			return true, fmt.Errorf("skaffold config %s from file %s imported multiple times with different profiles", configID, file)
		}
		return true, nil
	}
	appliedProfiles[key] = expected
	return false, nil
}
