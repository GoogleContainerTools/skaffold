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

package config

import (
	"errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

// Versions is an ordered list of all schema versions.
var Versions = []string{
	v1alpha1.Version,
	v1alpha2.Version,
	v1alpha3.Version,
	v1alpha4.Version,
}

var schemaVersions = map[string]func() util.VersionedConfig{
	v1alpha1.Version: func() util.VersionedConfig {
		return new(v1alpha1.SkaffoldConfig)
	},
	v1alpha2.Version: func() util.VersionedConfig {
		return new(v1alpha2.SkaffoldConfig)
	},
	v1alpha3.Version: func() util.VersionedConfig {
		return new(v1alpha3.SkaffoldConfig)
	},
	v1alpha4.Version: func() util.VersionedConfig {
		return new(v1alpha4.SkaffoldConfig)
	},
}

func GetConfig(contents []byte, useDefault bool) (util.VersionedConfig, error) {
	for _, version := range Versions {
		cfg := schemaVersions[version]()
		err := cfg.Parse(contents, useDefault)
		if cfg.GetVersion() == version {
			// Versions are same hence propagate the parse error.
			if err := yamltags.ProcessStruct(cfg); err != nil {
				return nil, err
			}
			return cfg, err
		}
	}

	return nil, errors.New("Unable to parse config")
}

type APIVersion struct {
	Version string `yaml:"apiVersion"`
}
