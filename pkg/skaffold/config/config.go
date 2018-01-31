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
	"bytes"
	"io"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

// SkaffoldConfig is the top level config object
// that is parsed from a skaffold.yaml
//
// APIVersion and Kind are currently reserved for future use.
type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Watch bool `yaml:"watch"`

	Build  BuildConfig  `yaml:"build"`
	Deploy DeployConfig `yaml:"deploy"`
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []Artifact `yaml:"artifacts"`
	TagPolicy string     `yaml:"tagPolicy"`
	BuildType `yaml:",inline"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild *LocalBuild `yaml:"local"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	Repository string `yaml:"repository"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	Name       string            `yaml:"name"`
	Parameters map[string]string `yaml:"parameters"`
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct{}

// Arifact represents items that need should be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName      string `yaml:"imageName"`
	DockerfilePath string `yaml:"dockerfilePath"`
	Workspace      string `yaml:"workspace"`
}

// Parse reads from an io.Reader and unmarshals the result into a SkaffoldConfig.
// The default config argument provides default values for the config,
// which can be overridden if present in the config file.
func Parse(defaultConfig *SkaffoldConfig, config io.Reader) (*SkaffoldConfig, error) {
	var b bytes.Buffer
	if _, err := b.ReadFrom(config); err != nil {
		return nil, errors.Wrap(err, "reading config")
	}

	var cfg SkaffoldConfig
	if defaultConfig != nil {
		cfg = *defaultConfig
	}
	if err := yaml.Unmarshal(b.Bytes(), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
