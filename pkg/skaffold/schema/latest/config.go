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
	Test     []*TestCase  `yaml:"test,omitempty" json:"test,omitempty"`
	Deploy   DeployConfig `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Profiles []Profile    `yaml:"profiles,omitempty" json:"profiles,omitempty"`
}

func (c *SkaffoldPipeline) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps
// +k8s:openapi-gen=true
type BuildConfig struct {
	//artifacts is a list of the actual images you're going to be building
	//you can include as many as you want here.
	Artifacts []*Artifact `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`

	// TagPolicy determines how skaffold is going to tag your images.
	// We provide a few strategies here, although you most likely won't need to care!
	// The policy can `gitCommit`, `sha256` or `envTemplate`.
	// If not specified, it defaults to `gitCommit: {}`.
	TagPolicy TagPolicy `yaml:"tagPolicy,omitempty" json:"tagPolicy,omitempty"`

	BuildType `yaml:",inline" json:",inline"`
}

// +k8s:openapi-gen=true
type TagPolicy struct {
	// Tag the image with the git commit of your current repository.
	GitCommit *GitTagger `yaml:"gitCommit,omitempty" yamltags:"oneOf=tag" json:"gitCommit,omitempty"`
	// Tag the image with the checksum of the built image (image id).
	Sha256 *ShaTagger `yaml:"sha256,omitempty" yamltags:"oneOf=tag" json:"sha256,omitempty"`

	// Tag the image with a configurable template string.
	// The template must be in the golang text/template syntax: https://golang.org/pkg/text/template/
	// The template is compiled and executed against the current environment,
	// with those variables injected:
	// <pre>
	//   IMAGE_NAME   |  Name of the image being built, as supplied in the artifacts section.
	//   DIGEST       |  Digest of the newly built image. For eg. `sha256:27ffc7f352665cc50ae3cbcc4b2725e36062f1b38c611b6f95d6df9a7510de23`.
	//   DIGEST_ALGO  |  Algorithm used by the digest: For eg. `sha256`.
	//   DIGEST_HEX   |  Digest of the newly built image. For eg. `27ffc7f352665cc50ae3cbcc4b2725e36062f1b38c611b6f95d6df9a7510de23`.
	// </pre>
	// Example:
	// <pre>
	// envTemplate:
	//  template: "{{.RELEASE}}-{{.IMAGE_NAME}}"
	// </pre>
	EnvTemplate *EnvTemplateTagger `yaml:"envTemplate,omitempty" yamltags:"oneOf=tag" json:"envTemplate,omitempty"`

	// Tag the image with the build timestamp.
	//  The format can be overridden with golang formats, see: https://golang.org/pkg/time/#Time.Format
	//    Default format is "2006-01-02_15-04-05.999_MST
	//  The timezone is by default the local timezone, this can be overridden, see https://golang.org/pkg/time/#Time.LoadLocation
	// dateTime:
	//   format: "2006-01-02"
	//   timezone: "UTC"
	DateTime *DateTimeTagger `yaml:"dateTime,omitempty" yamltags:"oneOf=tag" json:"dateTime,omitempty"`
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

	// This is where you'll put your specific builder configuration.
	// Valid builders are `local`, `googleCloudBuild` and `kaniko`.
	// Defaults to `local: {}`
	// Pushing the images can be skipped. If no value is specified, it'll default to
	// `true` on minikube or Docker for Desktop, for even faster build and deploy cycles.
	// `false` on other types of kubernetes clusters that require pushing the images.
	// skaffold defers to your ~/.docker/config for authentication information.
	// If you're using Google Container Registry, make sure that you have gcloud and
	// docker-credentials-helper-gcr configured correctly.
	//
	// By default, the local builder connects to the Docker daemon with Go code to build
	// images. If `useDockerCLI` is set, skaffold will simply shell out to the docker CLI.
	// `useBuildkit` can also be set to activate the experimental BuildKit feature.
	// <pre>
	// local:
	//   false by default for local clusters, true for remote clusters
	//   push: false
	//   useDockerCLI: false
	//   useBuildkit: false
	// </pre>
	Local *LocalBuild `yaml:"local,omitempty" yamltags:"oneOf=build" json:"local,omitempty"`

	// Docker artifacts can be built on Google Cloud Build. The projectId then needs
	// to be provided and the currently logged user should be given permissions to trigger
	// new builds on Cloud Build.
	// If the projectId is not provided, Skaffold will try to guess it from the image name.
	// For eg. If the artifact image name is gcr.io/myproject/image, then Skaffold will use
	// the `myproject` GCP project.
	// All the other parameters are also optional. The default values are listed here:
	// <pre>
	//  googleCloudBuild:
	//   projectId: YOUR_PROJECT
	//   diskSizeGb: 200
	//   machineType: "N1_HIGHCPU_8"|"N1_HIGHCPU_32"
	//   timeout: 10000s
	//   dockerImage: gcr.io/cloud-builders/docker
	// </pre>
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build" json:"googleCloudBuild,omitempty"`

	//Docker artifacts can be built on a Kubernetes cluster with Kaniko.
	//Exactly one buildContext must be specified to use kaniko
	//If localDir is specified, skaffold will mount sources directly via a emptyDir volume
	//If gcsBucket is specified, skaffold will send sources to the GCS bucket provided
	//Kaniko also needs access to a service account to push the final image.
	//See https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster
	//
	//kaniko:
	//  buildContext:
	//    gcsBucket: k8s-skaffold
	//    localDir: {}
	//  pullSecret: /a/secret/path/serviceaccount.json
	//  namespace: default
	//  timeout: 20m
	Kaniko *KanikoBuild `yaml:"kaniko,omitempty" yamltags:"oneOf=build" json:"kaniko,omitempty"`

	// Docker artifacts can be built on Azure Container Build.
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
// they should be built. Each artifact is of a given type among: `docker`, `bazel`, `jibMaven`, `jibGradle`.
// If not specified, it defaults to `docker: {}`.
// `image`, `context` and `sync` are defined alongside the type
// +k8s:openapi-gen=true
type Artifact struct {
	// The name of the image to be built.
	Image string `yaml:"image,omitempty" json:"image,omitempty"`
	// The path to your dockerfile context. Defaults to ".".
	Context string `yaml:"context,omitempty" json:"context,omitempty"`

	Sync map[string]string `yaml:"sync,omitempty" json:"sync,omitempty"`

	ArtifactType `yaml:",inline" json:",inline"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
// +k8s:openapi-gen=true
type Profile struct {
	Name   string       `yaml:"name,omitempty" json:"name,omitempty"`
	Build  BuildConfig  `yaml:"build,omitempty" json:"build,omitempty"`
	Test   []*TestCase  `yaml:"test,omitempty" json:"test,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty" json:"deploy,omitempty"`
}

// +k8s:openapi-gen=true
type ArtifactType struct {
	// docker defines a Dockerfile based artifact
	Docker *DockerArtifact `yaml:"docker,omitempty" yamltags:"oneOf=artifact" json:"docker,omitempty"`

	// bazel defines a Bazel based artifact
	Bazel *BazelArtifact `yaml:"bazel,omitempty" yamltags:"oneOf=artifact" json:"bazel,omitempty"`

	// jibMaven defines an artifact that is built with the JIB Maven plugin
	JibMaven *JibMavenArtifact `yaml:"jibMaven,omitempty" yamltags:"oneOf=artifact" json:"jibMaven,omitempty"`

	// jibGradle defines an artifact that is built with the JIB Gradle plugin

	JibGradle *JibGradleArtifact `yaml:"jibGradle,omitempty" yamltags:"oneOf=artifact" json:"jibGradle,omitempty"`
}

// Docker describes an artifact built from a Dockerfile,
// usually using `docker build`.
// +k8s:openapi-gen=true
type DockerArtifact struct {
	// Dockerfile's location relative to workspace. Defaults to "Dockerfile"
	Dockerfile string `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	// Key/value arguments passed to the docker build.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty" json:"buildArgs,omitempty"`
	// Images to consider as cache sources
	CacheFrom []string `yaml:"cacheFrom,omitempty" json:"cacheFrom,omitempty"`
	// Dockerfile target name to build.
	Target string `yaml:"target,omitempty" json:"target,omitempty"`
}

// +k8s:openapi-gen=true
type BazelArtifact struct {
	// bazel requires bazel CLI to be installed and the artifacts sources to
	// contain Bazel configuration files. Example:
	// <pre>
	// bazel:
	//  target: //:skaffold_example.tar
	// </pre>
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
