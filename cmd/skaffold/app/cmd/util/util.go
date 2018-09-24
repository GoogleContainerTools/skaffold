/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/apiversion"
	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func ParseConfig(filename string) (*config.SkaffoldConfig, error) {
	buf, err := util.ReadConfiguration(filename)
	if err != nil {
		return nil, errors.Wrap(err, "read skaffold config")
	}

	apiVersion := &config.APIVersion{}
	if err := yaml.Unmarshal(buf, apiVersion); err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	parsedVersion, err := apiversion.ParseVersion(apiVersion.Version)
	if err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	if parsedVersion.LT(config.LatestAPIVersion) {
		return nil, errors.New("Config version out of date: run `skaffold fix`")
	}

	if parsedVersion.GT(config.LatestAPIVersion) {
		return nil, errors.New("Config version is too new for this version of skaffold: upgrade skaffold")
	}

	cfg, err := config.GetConfig(buf, true)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold config")
	}

	// we already ensured that the versions match in the previous block,
	// so this type assertion is safe.
	return cfg.(*config.SkaffoldConfig), nil
}
