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

package latest

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const Version string = "skaffold/v1alpha4"

// NewSkaffoldPipeline creates a SkaffoldPipeline
func NewSkaffoldPipeline() util.VersionedConfig {
	return new(SkaffoldPipeline)
}

// SkaffoldPipeline defines the pipeline configuration for skaffold
// +k8s:openapi-gen=true
type SkaffoldPipeline struct {
	//apiVersion defines the version of the Pipeline API for skaffold
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	//kind should be SkaffoldPipeline
	Kind string `yaml:"kind" json:"kind"`

	Build    BuildConfig  `yaml:"build,omitempty" json:"build,omitempty"`
	Test     []TestCase   `yaml:"test,omitempty" json:"test,omitempty"`
	Deploy   DeployConfig `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Profiles []Profile    `yaml:"profiles,omitempty" json:"profiles,omitempty"`
}

func (c *SkaffoldPipeline) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps
// +k8s:openapi-gen=true
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	TagPolicy TagPolicy   `yaml:"tagPolicy,omitempty" json:"tagPolicy,omitempty"`
	BuildType `yaml:",inline" json:",inline"`
}

// TagPolicy contains all the configuration for the tagging step
// +k8s:openapi-gen=true
type TagPolicy struct {
	GitCommit   *GitTagger         `yaml:"gitCommit,omitempty" yamltags:"oneOf=tag" json:"gitCommit,omitempty"`
	Sha256      *ShaTagger         `yaml:"sha256,omitempty" yamltags:"oneOf=tag" json:"sha256,omitempty"`
	EnvTemplate *EnvTemplateTagger `yaml:"envTemplate,omitempty" yamltags:"oneOf=tag" json:"envTemplate,omitempty"`
	DateTime    *DateTimeTagger    `yaml:"dateTime,omitempty" yamltags:"oneOf=tag" json:"dateTime,omitempty"`
}

// ShaTagger contains the configuration for the SHA tagger.
// +k8s:openapi-gen=true
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
// +k8s:openapi-gen=true
type GitTagger struct{}

// EnvTemplateTagger contains the configuration for the envTemplate tagger.
// +k8s:openapi-gen=true
type EnvTemplateTagger struct {
	Template string `yaml:"template,omitempty" json:"template,omitempty"`
}

// DateTimeTagger contains the configuration for the DateTime tagger.
// +k8s:openapi-gen=true
type DateTimeTagger struct {
	Format   string `yaml:"format,omitempty" json:"format,omitempty"`
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
// +k8s:openapi-gen=true
type BuildType struct {
	Local               *LocalBuild          `yaml:"local,omitempty" yamltags:"oneOf=build" json:"local,omitempty"`
	GoogleCloudBuild    *GoogleCloudBuild    `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build" json:"googleCloudBuild,omitempty"`
	Kaniko              *KanikoBuild         `yaml:"kaniko,omitempty" yamltags:"oneOf=build" json:"kaniko,omitempty"`
	AzureContainerBuild *AzureContainerBuild `yaml:"azureContainerBuild,omitempty" yamltags:"oneOf=build" json:"azureContainerBuild,omitempty"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
// +k8s:openapi-gen=true
type LocalBuild struct {
	Push         *bool `yaml:"push,omitempty" json:"push,omitempty"`
	UseDockerCLI bool  `yaml:"useDockerCLI,omitempty" json:"useDockerCLI,omitempty"`
	UseBuildkit  bool  `yaml:"useBuildkit,omitempty" json:"useBuildkit,omitempty"`
}

// GoogleCloudBuild contains the fields needed to do a remote build on
// Google Cloud Build.
// +k8s:openapi-gen=true
type GoogleCloudBuild struct {
	ProjectID   string `yaml:"projectID,omitempty" json:"projectID,omitempty"`
	DiskSizeGb  int64  `yaml:"diskSizeGb,omitempty" json:"diskSizeGb,omitempty"`
	MachineType string `yaml:"machineType,omitempty" json:"machineType,omitempty"`
	Timeout     string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	DockerImage string `yaml:"dockerImage,omitempty" json:"dockerImage,omitempty"`
}

// LocalDir represents the local directory kaniko build context
// +k8s:openapi-gen=true
type LocalDir struct {
}

// KanikoBuildContext contains the different fields available to specify
// a kaniko build context
// +k8s:openapi-gen=true
type KanikoBuildContext struct {
	GCSBucket string    `yaml:"gcsBucket,omitempty" yamltags:"oneOf=buildContext" json:"gcsBucket,omitempty"`
	LocalDir  *LocalDir `yaml:"localDir,omitempty" yamltags:"oneOf=buildContext" json:"localDir,omitempty"`
}

// KanikoBuild contains the fields needed to do a on-cluster build using
// the kaniko image
// +k8s:openapi-gen=true
type KanikoBuild struct {
	BuildContext   *KanikoBuildContext `yaml:"buildContext,omitempty" json:"buildContext,omitempty"`
	PullSecret     string              `yaml:"pullSecret,omitempty" json:"pullSecret,omitempty"`
	PullSecretName string              `yaml:"pullSecretName,omitempty" json:"pullSecretName,omitempty"`
	Namespace      string              `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Timeout        string              `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Image          string              `yaml:"image,omitempty" json:"image,omitempty"`
}

// AzureContainerBuild contains the fields needed to do a build
// on Azure Container Registry
// +k8s:openapi-gen=true
type AzureContainerBuild struct {
	SubscriptionID string `yaml:"subscriptionID,omitempty" json:"subscriptionID,omitempty"`
	ClientID       string `yaml:"clientID,omitempty" json:"clientID,omitempty"`
	ClientSecret   string `yaml:"clientSecret,omitempty" json:"clientSecret,omitempty"`
	TenantID       string `yaml:"tenantID,omitempty" json:"tenantID,omitempty"`
}

// TestCase is a struct containing all the specified test
// configuration for an image.
// +k8s:openapi-gen=true
type TestCase struct {
	Image          string   `yaml:"image" json:"image"`
	StructureTests []string `yaml:"structureTests,omitempty" json:"structureTests,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps
// +k8s:openapi-gen=true
type DeployConfig struct {
	DeployType `yaml:",inline" json:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
// +k8s:openapi-gen=true
type DeployType struct {
	Helm      *HelmDeploy      `yaml:"helm,omitempty" yamltags:"oneOf=deploy" json:"helm,omitempty"`
	Kubectl   *KubectlDeploy   `yaml:"kubectl,omitempty" yamltags:"oneOf=deploy" json:"kubectl,omitempty"`
	Kustomize *KustomizeDeploy `yaml:"kustomize,omitempty" yamltags:"oneOf=deploy" json:"kustomize,omitempty"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
// +k8s:openapi-gen=true
type KubectlDeploy struct {
	Manifests       []string     `yaml:"manifests,omitempty" json:"manifests,omitempty"`
	RemoteManifests []string     `yaml:"remoteManifests,omitempty" json:"remoteManifests,omitempty"`
	Flags           KubectlFlags `yaml:"flags,omitempty" json:"flags,omitempty"`
}

// KubectlFlags describes additional options flags that are passed on the command
// line to kubectl either on every command (Global), on creations (Apply)
// or deletions (Delete).
// +k8s:openapi-gen=true
type KubectlFlags struct {
	Global []string `yaml:"global,omitempty" json:"global,omitempty"`
	Apply  []string `yaml:"apply,omitempty" json:"apply,omitempty"`
	Delete []string `yaml:"delete,omitempty" json:"delete,omitempty"`
}

// HelmDeploy contains the configuration needed for deploying with helm
// +k8s:openapi-gen=true
type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases,omitempty" json:"releases,omitempty"`
}

// KustomizeDeploy contains the configuration needed for deploying with kustomize.
// +k8s:openapi-gen=true
type KustomizeDeploy struct {
	Path  string       `yaml:"path,omitempty" json:"path,omitempty"`
	Flags KubectlFlags `yaml:"flags,omitempty" json:"flags,omitempty"`
}

// +k8s:openapi-gen=true
type HelmRelease struct {
	Name              string                 `yaml:"name,omitempty" json:"name,omitempty"`
	ChartPath         string                 `yaml:"chartPath,omitempty" json:"chartPath,omitempty"`
	ValuesFiles       []string               `yaml:"valuesFiles,omitempty" json:"valuesFiles,omitempty"`
	Values            map[string]string      `yaml:"values,omitempty,omitempty" json:"values,omitempty,omitempty"`
	Namespace         string                 `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Version           string                 `yaml:"version,omitempty" json:"version,omitempty"`
	SetValues         map[string]string      `yaml:"setValues,omitempty" json:"setValues,omitempty"`
	SetValueTemplates map[string]string      `yaml:"setValueTemplates,omitempty" json:"setValueTemplates,omitempty"`
	Wait              bool                   `yaml:"wait,omitempty" json:"wait,omitempty"`
	RecreatePods      bool                   `yaml:"recreatePods,omitempty" json:"recreatePods,omitempty"`
	Overrides         map[string]interface{} `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Packaged          *HelmPackaged          `yaml:"packaged,omitempty" json:"packaged,omitempty"`
	ImageStrategy     HelmImageStrategy      `yaml:"imageStrategy,omitempty" json:"imageStrategy,omitempty"`
}

// HelmPackaged represents parameters for packaging helm chart.
// +k8s:openapi-gen=true
type HelmPackaged struct {
	// Version sets the version on the chart to this semver version.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// AppVersion set the appVersion on the chart to this version
	AppVersion string `yaml:"appVersion,omitempty" json:"appVersion,omitempty"`
}

// +k8s:openapi-gen=true
type HelmImageStrategy struct {
	HelmImageConfig `yaml:",inline" json:",inline"`
}

// +k8s:openapi-gen=true
type HelmImageConfig struct {
	FQN  *HelmFQNConfig        `yaml:"fqn,omitempty" json:"fqn,omitempty"`
	Helm *HelmConventionConfig `yaml:"helm,omitempty" json:"helm,omitempty"`
}

// HelmFQNConfig represents image config to use the FullyQualifiedImageName as param to set
// +k8s:openapi-gen=true
type HelmFQNConfig struct {
	Property string `yaml:"property,omitempty" json:"property,omitempty"`
}

// HelmConventionConfig represents image config in the syntax of image.repository and image.tag
// +k8s:openapi-gen=true
type HelmConventionConfig struct {
}

// Artifact represents items that need to be built, along with the context in which
// they should be built.
// +k8s:openapi-gen=true
type Artifact struct {
	Image        string            `yaml:"image,omitempty" json:"image,omitempty"`
	Context      string            `yaml:"context,omitempty" json:"context,omitempty"`
	Sync         map[string]string `yaml:"sync,omitempty" json:"sync,omitempty"`
	ArtifactType `yaml:",inline" json:",inline"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
// +k8s:openapi-gen=true
type Profile struct {
	Name   string       `yaml:"name,omitempty" json:"name,omitempty"`
	Build  BuildConfig  `yaml:"build,omitempty" json:"build,omitempty"`
	Test   []TestCase   `yaml:"test,omitempty" json:"test,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty" json:"deploy,omitempty"`
}

// +k8s:openapi-gen=true
type ArtifactType struct {
	Docker    *DockerArtifact    `yaml:"docker,omitempty" yamltags:"oneOf=artifact" json:"docker,omitempty"`
	Bazel     *BazelArtifact     `yaml:"bazel,omitempty" yamltags:"oneOf=artifact" json:"bazel,omitempty"`
	JibMaven  *JibMavenArtifact  `yaml:"jibMaven,omitempty" yamltags:"oneOf=artifact" json:"jibMaven,omitempty"`
	JibGradle *JibGradleArtifact `yaml:"jibGradle,omitempty" yamltags:"oneOf=artifact" json:"jibGradle,omitempty"`
}

// Docker describes an artifact built from a Dockerfile,
// usually using `docker build`.
// +k8s:openapi-gen=true
type DockerArtifact struct {
	Dockerfile string             `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	BuildArgs  map[string]*string `yaml:"buildArgs,omitempty" json:"buildArgs,omitempty"`
	CacheFrom  []string           `yaml:"cacheFrom,omitempty" json:"cacheFrom,omitempty"`
	Target     string             `yaml:"target,omitempty" json:"target,omitempty"`
}

// Bazel describes an artifact built with Bazel.
// +k8s:openapi-gen=true
type BazelArtifact struct {
	Target string `yaml:"target,omitempty" json:"target,omitempty"`
}

// +k8s:openapi-gen=true
type JibMavenArtifact struct {
	// Only multi-module
	Module  string `yaml:"module" json:"module"`
	Profile string `yaml:"profile" json:"profile"`
}

// +k8s:openapi-gen=true
type JibGradleArtifact struct {
	// Only multi-module
	Project string `yaml:"project" json:"project"`
}

// Parse reads a SkaffoldPipeline from yaml.
func (c *SkaffoldPipeline) Parse(contents []byte, useDefaults bool) error {
	if err := yaml.UnmarshalStrict(contents, c); err != nil {
		return err
	}

	if useDefaults {
		if err := c.SetDefaultValues(); err != nil {
			return errors.Wrap(err, "applying default values")
		}
	}

	return nil
}
