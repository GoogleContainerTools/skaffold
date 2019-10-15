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

package v1beta3

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// !!! WARNING !!! This config version is already released, please DO NOT MODIFY the structs in this file.
const Version string = "skaffold/v1beta3"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}

type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build    BuildConfig  `yaml:"build,omitempty"`
	Test     TestConfig   `yaml:"test,omitempty"`
	Deploy   DeployConfig `yaml:"deploy,omitempty"`
	Profiles []Profile    `yaml:"profiles,omitempty"`
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`
	TagPolicy TagPolicy   `yaml:"tagPolicy,omitempty"`
	BuildType `yaml:",inline"`
}

// TagPolicy contains all the configuration for the tagging step
type TagPolicy struct {
	GitTagger         *GitTagger         `yaml:"gitCommit,omitempty" yamltags:"oneOf=tag"`
	ShaTagger         *ShaTagger         `yaml:"sha256,omitempty" yamltags:"oneOf=tag"`
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate,omitempty" yamltags:"oneOf=tag"`
	DateTimeTagger    *DateTimeTagger    `yaml:"dateTime,omitempty" yamltags:"oneOf=tag"`
}

// ShaTagger contains the configuration for the SHA tagger.
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
type GitTagger struct{}

// EnvTemplateTagger contains the configuration for the envTemplate tagger.
type EnvTemplateTagger struct {
	Template string `yaml:"template,omitempty"`
}

// DateTimeTagger contains the configuration for the DateTime tagger.
type DateTimeTagger struct {
	Format   string `yaml:"format,omitempty"`
	TimeZone string `yaml:"timezone,omitempty"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local,omitempty" yamltags:"oneOf=build"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build"`
	KanikoBuild      *KanikoBuild      `yaml:"kaniko,omitempty" yamltags:"oneOf=build"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	Push         *bool `yaml:"push,omitempty"`
	UseDockerCLI bool  `yaml:"useDockerCLI,omitempty"`
	UseBuildkit  bool  `yaml:"useBuildkit,omitempty"`
}

// GoogleCloudBuild contains the fields needed to do a remote build on
// Google Cloud Build.
type GoogleCloudBuild struct {
	ProjectID   string `yaml:"projectId,omitempty"`
	DiskSizeGb  int64  `yaml:"diskSizeGb,omitempty"`
	MachineType string `yaml:"machineType,omitempty"`
	Timeout     string `yaml:"timeout,omitempty"`
	DockerImage string `yaml:"dockerImage,omitempty"`
	MavenImage  string `yaml:"mavenImage,omitempty"`
	GradleImage string `yaml:"gradleImage,omitempty"`
}

// LocalDir represents the local directory kaniko build context
type LocalDir struct {
}

// KanikoBuildContext contains the different fields available to specify
// a kaniko build context
type KanikoBuildContext struct {
	GCSBucket string    `yaml:"gcsBucket,omitempty" yamltags:"oneOf=buildContext"`
	LocalDir  *LocalDir `yaml:"localDir,omitempty" yamltags:"oneOf=buildContext"`
}

// KanikoCache contains fields related to kaniko caching
type KanikoCache struct {
	Repo string `yaml:"repo,omitempty"`
}

// KanikoBuild contains the fields needed to do a on-cluster build using
// the kaniko image
type KanikoBuild struct {
	BuildContext    *KanikoBuildContext `yaml:"buildContext,omitempty"`
	Cache           *KanikoCache        `yaml:"cache,omitempty"`
	AdditionalFlags []string            `yaml:"flags,omitempty"`
	PullSecret      string              `yaml:"pullSecret,omitempty"`
	PullSecretName  string              `yaml:"pullSecretName,omitempty"`
	Namespace       string              `yaml:"namespace,omitempty"`
	Timeout         string              `yaml:"timeout,omitempty"`
	Image           string              `yaml:"image,omitempty"`
	DockerConfig    *DockerConfig       `yaml:"dockerConfig,omitempty"`
}

// DockerConfig contains information about the docker config.json to mount
type DockerConfig struct {
	Path       string `yaml:"path,omitempty"`
	SecretName string `yaml:"secretName,omitempty"`
}

type TestConfig []*TestCase

// TestCase is a struct containing all the specified test
// configuration for an image.
type TestCase struct {
	ImageName      string   `yaml:"image"`
	StructureTests []string `yaml:"structureTests,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy      *HelmDeploy      `yaml:"helm,omitempty" yamltags:"oneOf=deploy"`
	KubectlDeploy   *KubectlDeploy   `yaml:"kubectl,omitempty" yamltags:"oneOf=deploy"`
	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize,omitempty" yamltags:"oneOf=deploy"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests       []string     `yaml:"manifests,omitempty"`
	RemoteManifests []string     `yaml:"remoteManifests,omitempty"`
	Flags           KubectlFlags `yaml:"flags,omitempty"`
}

// KubectlFlags describes additional options flags that are passed on the command
// line to kubectl either on every command (Global), on creations (Apply)
// or deletions (Delete).
type KubectlFlags struct {
	Global []string `yaml:"global,omitempty"`
	Apply  []string `yaml:"apply,omitempty"`
	Delete []string `yaml:"delete,omitempty"`
}

// HelmDeploy contains the configuration needed for deploying with helm
type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases,omitempty"`
}

// KustomizeDeploy contains the configuration needed for deploying with kustomize.
type KustomizeDeploy struct {
	KustomizePath string       `yaml:"path,omitempty"`
	Flags         KubectlFlags `yaml:"flags,omitempty"`
}

type HelmRelease struct {
	Name              string             `yaml:"name,omitempty"`
	ChartPath         string             `yaml:"chartPath,omitempty"`
	ValuesFiles       []string           `yaml:"valuesFiles,omitempty"`
	Values            map[string]string  `yaml:"values,omitempty,omitempty"`
	Namespace         string             `yaml:"namespace,omitempty"`
	Version           string             `yaml:"version,omitempty"`
	SetValues         map[string]string  `yaml:"setValues,omitempty"`
	SetValueTemplates map[string]string  `yaml:"setValueTemplates,omitempty"`
	Wait              bool               `yaml:"wait,omitempty"`
	RecreatePods      bool               `yaml:"recreatePods,omitempty"`
	Overrides         util.HelmOverrides `yaml:"overrides,omitempty"`
	Packaged          *HelmPackaged      `yaml:"packaged,omitempty"`
	ImageStrategy     HelmImageStrategy  `yaml:"imageStrategy,omitempty"`
}

// HelmPackaged represents parameters for packaging helm chart.
type HelmPackaged struct {
	// Version sets the version on the chart to this semver version.
	Version string `yaml:"version,omitempty"`

	// AppVersion set the appVersion on the chart to this version
	AppVersion string `yaml:"appVersion,omitempty"`
}

type HelmImageStrategy struct {
	HelmImageConfig `yaml:",inline"`
}

type HelmImageConfig struct {
	HelmFQNConfig        *HelmFQNConfig        `yaml:"fqn,omitempty" yamltags:"oneOf=helmImageStrategy"`
	HelmConventionConfig *HelmConventionConfig `yaml:"helm,omitempty" yamltags:"oneOf=helmImageStrategy"`
}

// HelmFQNConfig represents image config to use the FullyQualifiedImageName as param to set
type HelmFQNConfig struct {
	Property string `yaml:"property,omitempty"`
}

// HelmConventionConfig represents image config in the syntax of image.repository and image.tag
type HelmConventionConfig struct {
}

// Artifact represents items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName    string            `yaml:"image,omitempty"`
	Workspace    string            `yaml:"context,omitempty"`
	Sync         map[string]string `yaml:"sync,omitempty"`
	ArtifactType `yaml:",inline"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
type Profile struct {
	Name   string       `yaml:"name,omitempty"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Test   TestConfig   `yaml:"test,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty"`
}

type ArtifactType struct {
	DockerArtifact    *DockerArtifact    `yaml:"docker,omitempty" yamltags:"oneOf=artifact"`
	BazelArtifact     *BazelArtifact     `yaml:"bazel,omitempty" yamltags:"oneOf=artifact"`
	JibMavenArtifact  *JibMavenArtifact  `yaml:"jibMaven,omitempty" yamltags:"oneOf=artifact"`
	JibGradleArtifact *JibGradleArtifact `yaml:"jibGradle,omitempty" yamltags:"oneOf=artifact"`
}

// DockerArtifact describes an artifact built from a Dockerfile,
// usually using `docker build`.
type DockerArtifact struct {
	DockerfilePath string             `yaml:"dockerfile,omitempty"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
	CacheFrom      []string           `yaml:"cacheFrom,omitempty"`
	Target         string             `yaml:"target,omitempty"`
}

// BazelArtifact describes an artifact built with Bazel.
type BazelArtifact struct {
	BuildTarget string   `yaml:"target,omitempty"`
	BuildArgs   []string `yaml:"args,omitempty"`
}

type JibMavenArtifact struct {
	// Only multi-module
	Module  string `yaml:"module"`
	Profile string `yaml:"profile"`
}

type JibGradleArtifact struct {
	// Only multi-module
	Project string `yaml:"project"`
}
