/*
Copyright 2018 Google LLC

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

package config

import (
	"errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

// Ordered list of all schema versions
var Versions = []string{v1alpha1.Version, v1alpha2.Version}

var schemaVersions = map[string]func([]byte, bool) (util.VersionedConfig, error){
	v1alpha1.Version: func(contents []byte, useDefault bool) (util.VersionedConfig, error) {
		config := new(v1alpha1.SkaffoldConfig)
		err := config.Parse(contents, useDefault)
		return config, err
	},
	v1alpha2.Version: func(contents []byte, useDefault bool) (util.VersionedConfig, error) {
		config := new(v1alpha2.SkaffoldConfig)
		err := config.Parse(contents, useDefault)
		return config, err
	},
}

func GetConfig(contents []byte, useDefault bool) (util.VersionedConfig, error) {
	for _, version := range Versions {
		if cfg, err := schemaVersions[version](contents, useDefault); err == nil {
			// successfully parsed, but make sure versions match
			if cfg.GetVersion() == version {
				return cfg, err
			}
		}
	}
	return nil, errors.New("Unable to parse config")
}

type ApiVersion struct {
	Version string `yaml:"apiVersion"`
}
