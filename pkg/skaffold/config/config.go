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

type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Watch bool `yaml:"watch"`

	Build  BuildConfig  `yaml:"build"`
	Deploy DeployConfig `yaml:"deploy"`
}

type BuildConfig struct {
	Artifacts []Artifact `yaml:"artifacts"`
	TagPolicy string     `yaml:"tagPolicy"`
	BuildType `yaml:",inline"`
}

type BuildType struct{}

type DeployConfig struct {
	Name       string            `yaml:"name"`
	Parameters map[string]string `yaml:"parameters"`
	DeployType `yaml:",inline"`
}

type DeployType struct{}

type Artifact struct {
	ImageName      string `yaml:"imageName"`
	DockerfilePath string `yaml:"dockerfilePath"`
	Workspace      string `yaml:"workspace"`
}

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
