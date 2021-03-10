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

package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/git"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
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

// record captures the state of referenced configs.
type record struct {
	appliedProfiles  map[string]string      // config -> list of applied profiles
	configNameToFile map[string]string      // configName -> file path
	cachedRepos      map[string]interface{} // git repo -> cache path or error
}

func newRecord() *record {
	return &record{appliedProfiles: make(map[string]string), configNameToFile: make(map[string]string), cachedRepos: make(map[string]interface{})}
}

// GetAllConfigs returns the list of all skaffold configurations parsed from the target config file in addition to all resolved dependency configs.
func GetAllConfigs(opts config.SkaffoldOptions) ([]*latest.SkaffoldConfig, error) {
	cOpts := configOpts{file: opts.ConfigurationFile, selection: nil, profiles: opts.Profiles, isRequired: false, isDependency: false}
	cfgs, err := getConfigs(cOpts, opts, newRecord())
	if err != nil {
		return nil, err
	}
	if len(cfgs) == 0 {
		if len(opts.ConfigurationFilter) > 0 {
			return nil, sErrors.BadConfigFilterErr(opts.ConfigurationFilter)
		}
		return nil, sErrors.ZeroConfigsParsedErr(opts.ConfigurationFile)
	}
	return cfgs, nil
}

// getConfigs recursively parses all configs and their dependencies in the specified `skaffold.yaml`
func getConfigs(cfgOpts configOpts, opts config.SkaffoldOptions, r *record) ([]*latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(cfgOpts.file, latest.Version)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, sErrors.MainConfigFileNotFoundErr(cfgOpts.file, err)
		}
		return nil, sErrors.ConfigParsingError(err)
	}

	if !util.IsURL(cfgOpts.file) && !filepath.IsAbs(cfgOpts.file) {
		cwd, _ := util.RealWorkDir()
		// convert `file` path to absolute value as it's used as a map key in several places.
		cfgOpts.file = filepath.Join(cwd, cfgOpts.file)
	}

	if len(parsed) == 0 {
		return nil, sErrors.ZeroConfigsParsedErr(cfgOpts.file)
	}
	logrus.Debugf("parsed %d configs from configuration file %s", len(parsed), cfgOpts.file)

	// validate that config names are unique if specified
	seen := make(map[string]bool)
	for _, cfg := range parsed {
		cfgName := cfg.(*latest.SkaffoldConfig).Metadata.Name
		if cfgName == "" {
			continue
		}
		if seen[cfgName] {
			return nil, sErrors.DuplicateConfigNamesInSameFileErr(cfgName, cfgOpts.file)
		}
		seen[cfgName] = true
	}

	var configs []*latest.SkaffoldConfig
	for i, cfg := range parsed {
		config := cfg.(*latest.SkaffoldConfig)
		processed, err := processEachConfig(config, cfgOpts, opts, r, i)
		if err != nil {
			return nil, err
		}
		configs = append(configs, processed...)
	}
	return configs, nil
}

// processEachConfig processes each parsed config by applying profiles and recursively processing its dependencies.
// The `index` parameter specifies the index of the current config in its `skaffold.yaml` file. We use the `index` instead of the config `metadata.name` property to uniquely identify each config since not all configs define `name`.
func processEachConfig(config *latest.SkaffoldConfig, cfgOpts configOpts, opts config.SkaffoldOptions, r *record, index int) ([]*latest.SkaffoldConfig, error) {
	// check that the same config name isn't repeated in multiple files.
	if config.Metadata.Name != "" {
		prevConfig, found := r.configNameToFile[config.Metadata.Name]
		if found && prevConfig != cfgOpts.file {
			return nil, sErrors.DuplicateConfigNamesAcrossFilesErr(config.Metadata.Name, prevConfig, cfgOpts.file)
		}
		r.configNameToFile[config.Metadata.Name] = cfgOpts.file
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
		return nil, sErrors.ConfigProfileActivationErr(config.Metadata.Name, cfgOpts.file, err)
	}
	if err := defaults.Set(config); err != nil {
		return nil, sErrors.ConfigSetDefaultValuesErr(config.Metadata.Name, cfgOpts.file, err)
	}
	// convert relative file paths to absolute for all configs that are not invoked explicitly. This avoids maintaining multiple root directory information since the dependency skaffold configs would have their own root directory.
	if cfgOpts.isDependency {
		if err := tags.MakeFilePathsAbsolute(config, filepath.Dir(cfgOpts.file)); err != nil {
			return nil, sErrors.ConfigSetAbsFilePathsErr(config.Metadata.Name, cfgOpts.file, err)
		}
	}

	sort.Strings(profiles)
	if revisit, err := checkRevisit(config, profiles, r.appliedProfiles, cfgOpts.file, required, index); revisit {
		return nil, err
	}

	var configs []*latest.SkaffoldConfig
	for _, d := range config.Dependencies {
		newOpts := configOpts{file: cfgOpts.file, profiles: filterActiveProfiles(d, profiles), isRequired: required, isDependency: cfgOpts.isDependency}
		depConfigs, err := processEachDependency(d, newOpts, opts, r)
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

// filterActiveProfiles selects the set of profiles to activate in the dependency config based on the current set of active profiles.
func filterActiveProfiles(d latest.ConfigDependency, profiles []string) []string {
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
	return depProfiles
}

// processEachDependency parses a config dependency with the calculated set of activated profiles.
func processEachDependency(d latest.ConfigDependency, cfgOpts configOpts, opts config.SkaffoldOptions, r *record) ([]*latest.SkaffoldConfig, error) {
	path := d.Path

	if d.GitRepo != nil {
		cachePath, err := cacheRepo(*d.GitRepo, opts, r)
		if err != nil {
			return nil, sErrors.ConfigParsingError(fmt.Errorf("caching remote dependency %s: %w", d.GitRepo.Repo, err))
		}
		path = cachePath
	}

	if path == "" {
		// empty path means configs in the same file
		path = cfgOpts.file
	}
	if !util.IsURL(path) {
		fi, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, sErrors.DependencyConfigFileNotFoundErr(path, cfgOpts.file, err)
			}
			return nil, sErrors.ConfigParsingError(fmt.Errorf("parsing dependencies for skaffold config %s: %w", cfgOpts.file, err))
		}
		if fi.IsDir() {
			path = filepath.Join(path, "skaffold.yaml")
		}
	}

	// if the current and previous configuration files are the same, then current config should be treated as a dependency config if the previous config was also a dependency config.
	// Otherwise the current config is always a dependency config if the file path is different than the previous.
	cfgOpts.isDependency = cfgOpts.isDependency || path != cfgOpts.file
	cfgOpts.file = path
	cfgOpts.selection = d.Names
	depConfigs, err := getConfigs(cfgOpts, opts, r)
	if err != nil {
		return nil, err
	}
	return depConfigs, nil
}

// cacheRepo downloads the referenced git repository to skaffold's cache if required and returns the path to the target configuration file in that repository.
func cacheRepo(g latest.GitInfo, opts config.SkaffoldOptions, r *record) (string, error) {
	key := fmt.Sprintf("%s@%s", g.Repo, g.Ref)
	if p, found := r.cachedRepos[key]; found {
		switch v := p.(type) {
		case string:
			return filepath.Join(v, g.Path), nil
		case error:
			return "", v
		default:
			logrus.Fatalf("unable to check download status of repo %s at ref %s", g.Repo, g.Ref)
			return "", nil
		}
	} else {
		p, err := git.SyncRepo(g, opts)
		if err != nil {
			r.cachedRepos[key] = err
			return "", err
		}
		r.cachedRepos[key] = p
		return filepath.Join(p, g.Path), nil
	}
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
			return true, sErrors.ConfigProfileConflictErr(configID, file)
		}
		return true, nil
	}
	appliedProfiles[key] = expected
	return false, nil
}
