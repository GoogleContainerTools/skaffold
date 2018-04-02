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
	yaml "gopkg.in/yaml.v2"
)

// SkaffoldConfig is the top level config object
// that is parsed from a skaffold.yaml
//
// APIVersion and Kind are currently reserved for future use.
type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build    BuildConfig  `yaml:"build"`
	Deploy   DeployConfig `yaml:"deploy"`
	Profiles []Profile    `yaml:"profiles"`
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`
	TagPolicy TagPolicy   `yaml:"tagPolicy,omitempty"`
	BuildType `yaml:",inline"`
}

// TagPolicy contains all the configuration for the tagging step
type TagPolicy struct {
	GitTagger         *GitTagger         `yaml:"git"`
	ShaTagger         *ShaTagger         `yaml:"sha256"`
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate"`
}

// ShaTagger contains the configuration for the SHA tagger.
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
type GitTagger struct{}

// EnvTemplateTagger contains the configuration for the envTemplate tagger.
type EnvTemplateTagger struct {
	Template string `yaml:"template"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	SkipPush *bool `yaml:"skipPush"`
}

type GoogleCloudBuild struct {
	ProjectID string `yaml:"projectId"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	Name       string `yaml:"name,omitempty"`
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy    *HelmDeploy    `yaml:"helm"`
	KubectlDeploy *KubectlDeploy `yaml:"kubectl"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests []string `yaml:"manifests,omitempty"`
}

type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases,omitempty"`
}

type HelmRelease struct {
	Name           string            `yaml:"name"`
	ChartPath      string            `yaml:"chartPath"`
	ValuesFilePath string            `yaml:"valuesFilePath"`
	Values         map[string]string `yaml:"values"`
	Namespace      string            `yaml:"namespace"`
	Version        string            `yaml:"version"`
}

// Artifact represents items that need should be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName      string             `yaml:"imageName"`
	DockerfilePath string             `yaml:"dockerfilePath,omitempty"`
	Workspace      string             `yaml:"workspace,omitempty"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
type Profile struct {
	Name   string       `yaml:"name"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty"`
}

// DefaultDevSkaffoldConfig is a partial set of defaults for the SkaffoldConfig
// when dev mode is specified.
// Each API is responsible for setting its own defaults that are not top level.
var DefaultDevSkaffoldConfig = &SkaffoldConfig{
	Build: BuildConfig{
		TagPolicy: TagPolicy{ShaTagger: &ShaTagger{}},
	},
}

// DefaultRunSkaffoldConfig is a partial set of defaults for the SkaffoldConfig
// when run mode is specified.
// Each API is responsible for setting its own defaults that are not top level.
var DefaultRunSkaffoldConfig = &SkaffoldConfig{
	Build: BuildConfig{
		TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
	},
}

// Parse reads from an io.Reader and unmarshals the result into a SkaffoldConfig.
// The default config argument provides default values for the config,
// which can be overridden if present in the config file.
func Parse(config []byte, defaultConfig *SkaffoldConfig) (*SkaffoldConfig, error) {
	if err := yaml.Unmarshal(config, defaultConfig); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}
