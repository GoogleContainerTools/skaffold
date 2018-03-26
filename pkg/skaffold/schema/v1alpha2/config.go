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

package v1alpha2

import (
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha1"

	yaml "gopkg.in/yaml.v2"
)

const Version string = "skaffold/v1alpha2"

type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build  BuildConfig  `yaml:"build"`
	Deploy DeployConfig `yaml:"deploy"`
}

func (config *SkaffoldConfig) GetVersion() string {
	return config.APIVersion
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	Name       string `yaml:"name,omitempty"`
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy    *v1alpha1.HelmDeploy `yaml:"helm,omitempty"`
	KubectlDeploy *KubectlDeploy       `yaml:"kubectl,omitempty"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests []string `yaml:"manifests"`
}

type BuildConfig struct {
	Artifacts          []*v1alpha1.Artifact `yaml:"artifacts"`
	TagPolicy          TagPolicy            `yaml:"tagPolicy,inline,omitempty"`
	v1alpha1.BuildType `yaml:",inline"`
}

// TagPolicy contains all the configuration for the tagging step
type TagPolicy struct {
	GitTagger *GitTagger `yaml:"git,omitempty"`
	ShaTagger *ShaTagger `yaml:"sha256,omitempty"`
}

// ShaTagger contains the configuration for the SHA tagger.
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
type GitTagger struct{}

var defaultDevSkaffoldConfig = &SkaffoldConfig{
	Build: BuildConfig{
		TagPolicy: TagPolicy{ShaTagger: &ShaTagger{}},
	},
}

var defaultRunSkaffoldConfig = &SkaffoldConfig{
	Build: BuildConfig{
		TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
	},
}

func (config *SkaffoldConfig) Parse(contents []byte, useDefault bool, mode bool) error {
	if useDefault {
		*config = *config.getDefaultForMode(mode)
	} else {
		*config = SkaffoldConfig{}
	}

	return yaml.Unmarshal(contents, config)
}

func (config *SkaffoldConfig) getDefaultForMode(dev bool) *SkaffoldConfig {
	if dev {
		return defaultDevSkaffoldConfig
	}
	return defaultRunSkaffoldConfig
}
