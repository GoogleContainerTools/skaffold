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

package latest

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	yamlpatch "github.com/krishicks/yaml-patch"
)

const Version string = "skaffold/v1beta5"

// NewSkaffoldPipeline creates a SkaffoldPipeline
func NewSkaffoldPipeline() util.VersionedConfig {
	return new(SkaffoldPipeline)
}

type SkaffoldPipeline struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	// Build describes how images are built.
	// **Required**
	Build BuildConfig `yaml:"build,omitempty"`

	// Test describes how images are tested.
	Test TestConfig `yaml:"test,omitempty"`

	// Deploy describes how images are deployed.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// Profiles (beta) has all the information which can be used to override any build,
	// test or deploy configuration.
	// The type of the deployment method can be `kubectl` (beta), `helm` (beta) or `kustomize` (beta).
	Profiles []Profile `yaml:"profiles,omitempty"`
}

func (c *SkaffoldPipeline) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps.
type BuildConfig struct {
	// Artifacts lists the images you're going to be building.
	// You can include as many as you want here.
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`

	// TagPolicy (beta) determines how Skaffold is going to tag images.
	// We provide a few strategies here, although you most likely won't need to care!
	// The policy can be `gitCommit` (beta), `sha256` (beta), `envTemplate` (beta) or `dateTime` (beta).
	// If not specified, it defaults to `gitCommit: {}`.
	TagPolicy TagPolicy `yaml:"tagPolicy,omitempty"`

	ExecutionEnvironment *ExecutionEnvironment `yaml:"executionEnvironment,omitempty"`

	BuildType `yaml:",inline"`
}

type ExecEnvironment string

// ExecutionEnvironment is the environment in which the build should run (ex. local or in-cluster, etc.)
type ExecutionEnvironment struct {
	Name       ExecEnvironment        `yaml:"name,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

// BuilderPlugin contains all fields necessary for specifying a build plugin
type BuilderPlugin struct {
	// Name of the build plugin
	Name string `yaml:"name,omitempty"`
	// Properties associated with the plugin
	Properties map[string]interface{} `yaml:"properties,omitempty"`
	Contents   []byte                 `yaml:",omitempty"`
}

// TagPolicy contains all the configuration for the tagging step
type TagPolicy struct {
	// GitTagger tags images with the git tag or git commit of the artifact workspace directory.
	GitTagger *GitTagger `yaml:"gitCommit,omitempty" yamltags:"oneOf=tag"`

	// ShaTagger tags images with their sha256 digest.
	ShaTagger *ShaTagger `yaml:"sha256,omitempty" yamltags:"oneOf=tag"`

	// EnvTemplateTagger tags images with a configurable template string.
	// The template must be in the golang text/template syntax: https://golang.org/pkg/text/template/
	// The template is compiled and executed against the current environment,
	// with those variables injected:
	//   IMAGE_NAME   |  Name of the image being built, as supplied in the artifacts section.
	// For example: "{{.RELEASE}}-{{.IMAGE_NAME}}"
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate,omitempty" yamltags:"oneOf=tag"`

	// DateTimeTagger tags images with the build timestamp.
	// The format can be overridden with golang formats, see: https://golang.org/pkg/time/#Time.Format
	// Default format is "2006-01-02_15-04-05.999_MST
	// The timezone is by default the local timezone, this can be overridden, see https://golang.org/pkg/time/#Time.LoadLocation
	// For example:
	// dateTime:
	//   format: "2006-01-02"
	//   timezone: "UTC"
	DateTimeTagger *DateTimeTagger `yaml:"dateTime,omitempty" yamltags:"oneOf=tag"`
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
	// LocalBuild describes how to do a build on the local docker daemon
	// and optionally push to a repository.
	LocalBuild *LocalBuild `yaml:"local,omitempty" yamltags:"oneOf=build"`

	// GoogleCloudBuild describes how to do a remote build on Google Cloud Build.
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build"`

	// KanikoBuild describes how to do an on-cluster build using
	// the kaniko image.
	KanikoBuild *KanikoBuild `yaml:"kaniko,omitempty" yamltags:"oneOf=build"`
}

// LocalBuild describes how to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	// Push should images be pushed to a registry.
	// Default: `false` for local clusters, `true` for remote clusters.
	Push *bool `yaml:"push,omitempty"`

	// UseDockerCLI use `docker` command-line interface instead of Docker Engine APIs.
	UseDockerCLI bool `yaml:"useDockerCLI,omitempty"`

	// UseBuildkit use BuildKit to build Docker images.
	UseBuildkit bool `yaml:"useBuildkit,omitempty"`
}

// GoogleCloudBuild describes how to do a remote build on
// [Google Cloud Build](https://cloud.google.com/cloud-build/docs/).
// Docker and Jib artifacts can be built on Cloud Build. The `projectId` needs
// to be provided and the currently logged in user should be given permissions to trigger
// new builds.
type GoogleCloudBuild struct {
	// ProjectID the ID of your Google Cloud Platform Project.
	// If the projectId is not provided, Skaffold will guess it from the image name.
	// For example, if the artifact image name is `gcr.io/myproject/image`, then Skaffold
	// will use the `myproject` GCP project.
	ProjectID string `yaml:"projectId,omitempty"`

	// DiskSizeGb the disk size of the VM that runs the build.
	// See [Cloud Build API Reference: Build Options](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions)
	// for more information.
	DiskSizeGb int64 `yaml:"diskSizeGb,omitempty"`

	// MachineType the type of the VM that runs the build.
	// See [Cloud Build API Reference: Build Options](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions)
	// for more information.
	MachineType string `yaml:"machineType,omitempty"`

	// Timeout the amount of time (in seconds) that this build should be allowed to run.
	// See [Cloud Build API Reference: Resource/Build](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#resource-build)
	// for more information.
	Timeout string `yaml:"timeout,omitempty"`

	// DockerImage the name of the image that will run a docker build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders)
	// for more information.
	// Defaults to `gcr.io/cloud-builders/docker`.
	DockerImage string `yaml:"dockerImage,omitempty"`

	// MavenImage the name of the image that will run a maven build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders)
	// for more information.
	// Defaults to `gcr.io/cloud-builders/mvn`.
	MavenImage string `yaml:"mavenImage,omitempty"`

	// GradleImage the name of the image that will run a gradle build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders)
	// for more information.
	// Defaults to `gcr.io/cloud-builders/gradle`.
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

// KanikoBuild describes how to do a on-cluster build using the kaniko image.
type KanikoBuild struct {
	// BuildContext the Kaniko build context: `gcsBucket` or `localDir`.
	// Defaults to `localDir`.
	BuildContext *KanikoBuildContext `yaml:"buildContext,omitempty"`

	Cache *KanikoCache `yaml:"cache,omitempty"`

	AdditionalFlags []string `yaml:"flags,omitempty"`

	// PullSecret the path to the secret key file.
	// See [Kaniko Documentation: Running Kaniko in a Kubernetes cluster](https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster)
	// for more information.
	PullSecret string `yaml:"pullSecret,omitempty"`

	// PullSecretName the name of the Kubernetes secret for pulling the files
	// from the build context and pushing the final image.
	// Defaults to `kaniko-secret`.
	PullSecretName string `yaml:"pullSecretName,omitempty"`

	// Namespace the Kubernetes namespace.
	// Defaults to current namespace in Kubernetes configuration.
	Namespace string `yaml:"namespace,omitempty"`

	// Timeout the amount of time (in seconds) that this build should be allowed to run.
	// Defaults to 20 minutes (`20m`).
	Timeout string `yaml:"timeout,omitempty"`

	// Image used bu the Kaniko pod.
	// Defaults to the latest released version of `gcr.io/kaniko-project/executor`
	Image string `yaml:"image,omitempty"`

	// DockerConfig
	DockerConfig *DockerConfig `yaml:"dockerConfig,omitempty"`
}

// DockerConfig contains information about the docker config.json to mount
type DockerConfig struct {
	// Path path to the docker `config.json`
	Path string `yaml:"path,omitempty"`

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
	HelmDeploy *HelmDeploy `yaml:"helm,omitempty" yamltags:"oneOf=deploy"`

	// KubectlDeploy uses a client side `kubectl apply` to apply the manifests to the cluster.
	// You'll need a kubectl CLI version installed that's compatible with your cluster.
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty" yamltags:"oneOf=deploy"`

	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize,omitempty" yamltags:"oneOf=deploy"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	// Manifests lists the Kubernetes yaml or json manifests.
	// Defaults to `[\"k8s/*.yaml\"]`.
	Manifests []string `yaml:"manifests,omitempty"`

	// RemoteManifests lists Kubernetes Manifests in remote clusters.
	RemoteManifests []string `yaml:"remoteManifests,omitempty"`

	// Flags additional flags to pass to `kubectl`. You can specify three types of flags: <ul><li>`global`: flags that apply to every command.</li><li>`apply`: flags that apply to creation commands.</li><li>`delete`: flags that apply to deletion commands.</li><ul>
	Flags KubectlFlags `yaml:"flags,omitempty"`
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
	// Releases a list of Helm releases.
	// **Required**
	Releases []HelmRelease `yaml:"releases,omitempty"`
}

// KustomizeDeploy contains the configuration needed for deploying with kustomize.
type KustomizeDeploy struct {
	// KustomizePath path to Kustomization files.
	// Default to `.` (current directory).
	KustomizePath string `yaml:"path,omitempty"`

	// Flags additional flags to pass to `kubectl`.
	// You can specify three types of flags: <ul><li>`global`: flags that apply to every command.</li><li>`apply`: flags that apply to creation commands.</li><li>`delete`: flags that apply to deletion commands.</li><ul>
	Flags KubectlFlags `yaml:"flags,omitempty"`
}

type HelmRelease struct {
	// Name the name of the Helm release.
	// **Required**
	Name string `yaml:"name,omitempty"`

	// ChartPath the path to the Helm chart.
	// **Required**
	ChartPath string `yaml:"chartPath,omitempty"`

	// ValuesFiles the paths to the Helm `values` files".
	ValuesFiles []string `yaml:"valuesFiles,omitempty"`

	// Values a list of key-value pairs supplementing the Helm `values` file".
	Values map[string]string `yaml:"values,omitempty,omitempty"`

	// Namespace the Kubernetes namespace.
	Namespace string `yaml:"namespace,omitempty"`

	// Version the version of the chart.
	Version string `yaml:"version,omitempty"`

	// SetValues a list of key-value pairs.
	// If present, Skaffold will send `--set` flag to Helm CLI and append all pairs after the flag.
	SetValues map[string]string `yaml:"setValues,omitempty"`

	// SetValueTemplates a list of key-value pairs.
	// If present, Skaffold will try to parse the value part of each key-value pair using
	// environment variables in the system, then send `--set` flag to Helm CLI and append
	// all parsed pairs after the flag.
	SetValueTemplates map[string]string `yaml:"setValueTemplates,omitempty"`

	// Wait if `true`, Skaffold will send `--wait` flag to Helm CLI.
	// Defaults to `false`.
	Wait bool `yaml:"wait,omitempty"`

	// RecreatePods if `true`, Skaffold will send `--recreate-pods` flag to Helm CLI.
	// Defaults to `false`.
	RecreatePods bool `yaml:"recreatePods,omitempty"`

	SkipBuildDependencies bool `yaml:"skipBuildDependencies,omitempty"`

	// Overrides a list of key-value pairs.
	// If present, Skaffold will build a Helm `values` file that overrides
	// the original and use it to call Helm CLI (`--f` flag).
	Overrides map[string]interface{} `yaml:"overrides,omitempty"`

	// Packaged packages the chart (`helm package`).
	// Includes two fields: <ul><li>`version`: Version of the chart.</li><li>`appVersion`: Version of the app.</li></ul>.
	Packaged *HelmPackaged `yaml:"packaged,omitempty"`

	// ImageStrategy add image configurations to the Helm `values` file.
	// Includes one of the two following fields: <ul><li> `fqn`: The image configuration uses the syntax `IMAGE-NAME=IMAGE-REPOSITORY:IMAGE-TAG`. </li><li>`helm`: The image configuration uses the syntax `IMAGE-NAME.repository=IMAGE-REPOSITORY, IMAGE-NAME.tag=IMAGE-TAG`.</li></ul>
	ImageStrategy HelmImageStrategy `yaml:"imageStrategy,omitempty"`
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
	HelmFQNConfig        *HelmFQNConfig        `yaml:"fqn,omitempty"`
	HelmConventionConfig *HelmConventionConfig `yaml:"helm,omitempty"`
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
	// ImageName name of the image to be built.
	ImageName string `yaml:"image,omitempty"`

	// Workspace directory where the artifact's sources are to be found.
	// Defaults to `.`.
	Workspace string `yaml:"context,omitempty"`

	// Skaffold can sync local files with remote pods (alpha) instead
	// of rebuilding the whole artifact's image. This is a mapping
	// of local files to sync to remote folders.
	// For example:
	// ```
	// sync:
	//   '*.py': .
	// ```
	Sync map[string]string `yaml:"sync,omitempty"`

	ArtifactType `yaml:",inline"`

	// The plugin used to build this artifact
	BuilderPlugin *BuilderPlugin `yaml:"plugin,omitempty"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
type Profile struct {
	// Name unique profile name.
	Name string `yaml:"name,omitempty"`

	Build   BuildConfig     `yaml:"build,omitempty"`
	Test    TestConfig      `yaml:"test,omitempty"`
	Deploy  DeployConfig    `yaml:"deploy,omitempty"`
	Patches yamlpatch.Patch `yaml:"patches,omitempty"`

	// Activation criteria by which a profile can be auto-activated.
	// This can be based on Environment Variables, the current Kubernetes
	// context name, or depending on which Skaffold command is running.
	Activation []Activation `yaml:"activation,omitempty"`
}

// Activation defines criteria by which a profile is auto-activated.
type Activation struct {
	// Env holds a key=value pair. The profile is auto-activated if an Environment
	// Variable `key` has value `value`.
	// For example: `ENV=production` or `DEBUG=true`
	Env string `yaml:"env,omitempty"`
	// KubeContext defines for which Kubernetes context, a profile is auto-activated.
	// For example: `minikube` or `docker-desktop`.
	KubeContext string `yaml:"kubeContext,omitempty"`
	// Command defines for which Skaffold command, a profile is auto-activated.
	// For example: `run` or `dev`.
	Command string `yaml:"command,omitempty"`
}

type ArtifactType struct {
	// DockerArtifact describes an artifact built from a Dockerfile,
	// usually using `docker build`.
	DockerArtifact *DockerArtifact `yaml:"docker,omitempty" yamltags:"oneOf=artifact"`

	// BazelArtifact requires bazel CLI to be installed and the artifacts sources to
	// contain Bazel configuration files.
	BazelArtifact *BazelArtifact `yaml:"bazel,omitempty" yamltags:"oneOf=artifact"`

	// JibMavenArtifact builds containers using the Jib plugin for Maven.
	JibMavenArtifact *JibMavenArtifact `yaml:"jibMaven,omitempty" yamltags:"oneOf=artifact"`

	// JibGradleArtifact builds containers using the Jib plugin for Gradle.
	JibGradleArtifact *JibGradleArtifact `yaml:"jibGradle,omitempty" yamltags:"oneOf=artifact"`
}

// DockerArtifact describes an artifact built from a Dockerfile,
// usually using `docker build`.
type DockerArtifact struct {
	// DockerfilePath locates the Dockerfile relative to workspace.
	// Defaults to "Dockerfile"
	DockerfilePath string `yaml:"dockerfile,omitempty"`

	// BuildArgs arguments passed to the docker build.
	// For eample:
	// buildArgs:
	//   key1: "value1"
	//   key2: "value2"
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`

	// CacheFrom lists the Docker images to consider as cache sources.
	// for example: ["golang:1.10.1-alpine3.7", "alpine:3.7"]
	CacheFrom []string `yaml:"cacheFrom,omitempty"`

	// Target Dockerfile target name to build.
	Target string `yaml:"target,omitempty"`
}

// BazelArtifact describes an artifact built with Bazel.
type BazelArtifact struct {
	// BuildTarget the `bazel build` target to run
	// For example: "//:skaffold_example.tar"
	BuildTarget string `yaml:"target,omitempty"`

	// BuildArgs additional args to pass to `bazel build`.
	// For example: ["arg1", "arg2"]
	BuildArgs []string `yaml:"args,omitempty"`
}

// JibMavenArtifact builds containers using the Jib plugin for Maven.
type JibMavenArtifact struct {
	// Module selects which maven module to build, for a multimodule project.
	Module string `yaml:"module"`

	// Profile selects which maven profile to activate.
	Profile string `yaml:"profile"`
}

// JibGradleArtifact builds containers using the Jib plugin for Gradle.
type JibGradleArtifact struct {
	// Project selects which gradle project to build.
	Project string `yaml:"project"`
}
