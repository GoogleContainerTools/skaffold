/*
Copyright 2019 The Skaffold Authors

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

package v1alpha1

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// !!! WARNING !!! This config version is already released, please DO NOT MODIFY the structs in this file.
const Version string = "skaffold/v1alpha1"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}

// SkaffoldConfig is the top level config object
// that is parsed from a skaffold.yaml
type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build  BuildConfig  `yaml:"build"`
	Deploy DeployConfig `yaml:"deploy"`
}

func (config *SkaffoldConfig) GetVersion() string {
	return config.APIVersion
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts"`
	TagPolicy string      `yaml:"tagPolicy,omitempty"`
	BuildType `yaml:",inline"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local,omitempty" yamltags:"oneOf=build"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	SkipPush *bool `yaml:"skipPush,omitempty"`
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
	HelmDeploy    *HelmDeploy    `yaml:"helm,omitempty" yamltags:"oneOf=deploy"`
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty" yamltags:"oneOf=deploy"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests []Manifest `yaml:"manifests"`
}

type Manifest struct {
	Paths      []string          `yaml:"paths"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
}

type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases"`
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
	Workspace      string             `yaml:"workspace"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
}
