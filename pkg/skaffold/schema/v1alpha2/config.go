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

package v1alpha2

import (
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const Version string = "skaffold/v1alpha2"

type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build    BuildConfig  `yaml:"build,omitempty"`
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
	GitTagger         *GitTagger         `yaml:"gitCommit" yamltags:"oneOf=tag"`
	ShaTagger         *ShaTagger         `yaml:"sha256" yamltags:"oneOf=tag"`
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate" yamltags:"oneOf=tag"`
	DateTimeTagger    *DateTimeTagger    `yaml:"dateTime" yamltags:"oneOf=tag"`
}

// ShaTagger contains the configuration for the SHA tagger.
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
type GitTagger struct{}

// EnvTemplateTagger contains the configuration for the envTemplate tagger.
type EnvTemplateTagger struct {
	Template string `yaml:"template"`
}

// DateTimeTagger contains the configuration for the DateTime tagger.
type DateTimeTagger struct {
	Format   string `yaml:"format,omitempty"`
	TimeZone string `yaml:"timezone,omitempty"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local" yamltags:"oneOf=build"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild" yamltags:"oneOf=build"`
	KanikoBuild      *KanikoBuild      `yaml:"kaniko" yamltags:"oneOf=build"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	SkipPush     *bool `yaml:"skipPush"`
	UseDockerCLI bool  `yaml:"useDockerCLI"`
	UseBuildkit  bool  `yaml:"useBuildkit"`
}

// GoogleCloudBuild contains the fields needed to do a remote build on
// Google Cloud Build.
type GoogleCloudBuild struct {
	ProjectID   string `yaml:"projectId"`
	DiskSizeGb  int64  `yaml:"diskSizeGb,omitempty"`
	MachineType string `yaml:"machineType,omitempty"`
	Timeout     string `yaml:"timeout,omitempty"`
	DockerImage string `yaml:"dockerImage,omitempty"`
}

// KanikoBuild contains the fields needed to do a on-cluster build using
// the kaniko image
type KanikoBuild struct {
	ContextType    `yaml:",inline"`
	Env            []*Env         `yaml:"env,omitempty"`
	Volumes        []*Volume      `yaml:"volumes,omitempty"`
	VolumeMounts   []*VolumeMount `yaml:"volumeMounts,omitempty"`
	PullSecret     string         `yaml:"pullSecret,omitempty"`
	PullSecretName string         `yaml:"pullSecretName,omitempty"`
	Namespace      string         `yaml:"namespace,omitempty"`
	Timeout        string         `yaml:"timeout,omitempty"`
}

type ContextType struct {
	GcsContext      *GcsContext      `yaml:"gcs" yamltags:"oneOf=context"`
	LocalDirContext *LocalDirContext `yaml:"localDir" yamltags:"oneOf=context"`
	S3Context       *S3Context       `yaml:"s3" yamltags:"oneOf=context"`
}

type Env struct {
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}

type VolumeMount struct {
	Name      string `yaml:"name,omitempty"`
	MountPath string `yaml:"mountPath,omitempty"`
}

type Volume struct {
	Name     string `yaml:"name,omitempty"`
	HostPath string `yaml:"hostPath,omitempty"`
	Secret   string `yaml:"secret,omitempty"`
}

type GcsContext struct {
	GCSBucket string `yaml:"bucket,omitempty"`
}

type S3Context struct {
	S3Bucket string `yaml:"bucket,omitempty"`
	Region string `yaml:"region,omitempty"`
}

type LocalDirContext struct {
	Path string `yaml:"path,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy      *HelmDeploy      `yaml:"helm" yamltags:"oneOf=deploy"`
	KubectlDeploy   *KubectlDeploy   `yaml:"kubectl" yamltags:"oneOf=deploy"`
	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize" yamltags:"oneOf=deploy"`
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

type KustomizeDeploy struct {
	KustomizePath string       `yaml:"kustomizePath,omitempty"`
	Flags         KubectlFlags `yaml:"flags,omitempty"`
}

type HelmRelease struct {
	Name              string                 `yaml:"name"`
	ChartPath         string                 `yaml:"chartPath"`
	ValuesFilePath    string                 `yaml:"valuesFilePath"`
	Values            map[string]string      `yaml:"values,omitempty"`
	Namespace         string                 `yaml:"namespace"`
	Version           string                 `yaml:"version"`
	SetValues         map[string]string      `yaml:"setValues"`
	SetValueTemplates map[string]string      `yaml:"setValueTemplates"`
	Wait              bool                   `yaml:"wait"`
	RecreatePods      bool                   `yaml:"recreatePods"`
	Overrides         map[string]interface{} `yaml:"overrides"`
	Packaged          *HelmPackaged          `yaml:"packaged"`
	ImageStrategy     HelmImageStrategy      `yaml:"imageStrategy"`
}

// HelmPackaged represents parameters for packaging helm chart.
type HelmPackaged struct {
	// Version sets the version on the chart to this semver version.
	Version string `yaml:"version"`

	// AppVersion set the appVersion on the chart to this version
	AppVersion string `yaml:"appVersion"`
}

type HelmImageStrategy struct {
	HelmImageConfig `yaml:",inline"`
}

type HelmImageConfig struct {
	HelmFQNConfig        *HelmFQNConfig        `yaml:"fqn"`
	HelmConventionConfig *HelmConventionConfig `yaml:"helm"`
}

// HelmFQNConfig represents image config to use the FullyQualifiedImageName as param to set
type HelmFQNConfig struct {
	Property string `yaml:"property"`
}

// HelmConventionConfig represents image config in the syntax of image.repository and image.tag
type HelmConventionConfig struct {
}

// Artifact represents items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName    string `yaml:"imageName"`
	Workspace    string `yaml:"workspace,omitempty"`
	ArtifactType `yaml:",inline"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
type Profile struct {
	Name   string       `yaml:"name"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty"`
}

type ArtifactType struct {
	DockerArtifact *DockerArtifact `yaml:"docker" yamltags:"oneOf=artifact"`
	BazelArtifact  *BazelArtifact  `yaml:"bazel" yamltags:"oneOf=artifact"`
}

type DockerArtifact struct {
	DockerfilePath string             `yaml:"dockerfilePath,omitempty"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
	CacheFrom      []string           `yaml:"cacheFrom,omitempty"`
	Target         string             `yaml:"target,omitempty"`
}

type BazelArtifact struct {
	BuildTarget string `yaml:"target"`
}

// Parse reads a SkaffoldConfig from yaml.
func (c *SkaffoldConfig) Parse(contents []byte, useDefaults bool) error {
	if err := yaml.UnmarshalStrict(contents, c); err != nil {
		return err
	}

	if useDefaults {
		if err := c.setDefaultValues(); err != nil {
			return errors.Wrap(err, "applying default values")
		}
	}

	return nil
}
