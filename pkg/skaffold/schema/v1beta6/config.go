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

package v1beta6

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
)

// !!! WARNING !!! This config version is already released, please DO NOT MODIFY the structs in this file.
const Version string = "skaffold/v1beta6"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}

// SkaffoldConfig describes a Skaffold pipeline.
type SkaffoldConfig struct {
	// APIVersion is the version of the configuration.
	APIVersion string `yaml:"apiVersion"`

	// Kind is always `Config`. Defaults to `Config`.
	Kind string `yaml:"kind"`

	// Build describes how images are built.
	Build BuildConfig `yaml:"build,omitempty"`

	// Test describes how images are tested.
	Test []*TestCase `yaml:"test,omitempty"`

	// Deploy describes how images are deployed.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// Profiles *beta* can override be used to `build`, `test` or `deploy` configuration.
	Profiles []Profile `yaml:"profiles,omitempty"`
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps.
type BuildConfig struct {
	// Artifacts lists the images you're going to be building.
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`

	// TagPolicy *beta* determines how images are tagged.
	// A few strategies are provided here, although you most likely won't need to care!
	// If not specified, it defaults to `gitCommit: {}`.
	TagPolicy TagPolicy `yaml:"tagPolicy,omitempty"`

	// ExecutionEnvironment is the environment in which the build
	// should run. Possible values: googleCloudBuild.
	ExecutionEnvironment *ExecutionEnvironment `yaml:"executionEnvironment,omitempty"`

	BuildType `yaml:",inline"`
}

// ExecEnvironment is the name of an execution environment.
type ExecEnvironment string

// ExecutionEnvironment is the environment in which the build should run (ex. local or in-cluster, etc.).
type ExecutionEnvironment struct {
	// Name is the name of the environment.
	Name ExecEnvironment `yaml:"name,omitempty"`

	// Properties are key-value pairs passed to the environment.
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

// BuilderPlugin contains all fields necessary for specifying a build plugin.
type BuilderPlugin struct {
	// Name is the name of the build plugin.
	Name string `yaml:"name,omitempty"`

	// Properties are key-value pairs passed to the plugin.
	Properties map[string]interface{} `yaml:"properties,omitempty"`

	// Contents
	Contents []byte `yaml:",omitempty"`
}

// TagPolicy contains all the configuration for the tagging step.
type TagPolicy struct {
	// GitTagger *beta* tags images with the git tag or commit of the artifact's workspace.
	GitTagger *GitTagger `yaml:"gitCommit,omitempty" yamltags:"oneOf=tag"`

	// ShaTagger *beta* tags images with their sha256 digest.
	ShaTagger *ShaTagger `yaml:"sha256,omitempty" yamltags:"oneOf=tag"`

	// EnvTemplateTagger *beta* tags images with a configurable template string.
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate,omitempty" yamltags:"oneOf=tag"`

	// DateTimeTagger *beta* tags images with the build timestamp.
	DateTimeTagger *DateTimeTagger `yaml:"dateTime,omitempty" yamltags:"oneOf=tag"`
}

// ShaTagger *beta* tags images with their sha256 digest.
type ShaTagger struct{}

// GitTagger *beta* tags images with the git tag or commit of the artifact's workspace.
type GitTagger struct{}

// EnvTemplateTagger *beta* tags images with a configurable template string.
type EnvTemplateTagger struct {
	// Template used to produce the image name and tag.
	// See golang [text/template](https://golang.org/pkg/text/template/).
	// The template is executed against the current environment,
	// with those variables injected:
	//   IMAGE_NAME   |  Name of the image being built, as supplied in the artifacts section.
	// For example: `{{.RELEASE}}-{{.IMAGE_NAME}}`.
	Template string `yaml:"template,omitempty" yamltags:"required"`
}

// DateTimeTagger *beta* tags images with the build timestamp.
type DateTimeTagger struct {
	// Format formats the date and time.
	// See [#Time.Format](https://golang.org/pkg/time/#Time.Format).
	// Defaults to `2006-01-02_15-04-05.999_MST`.
	Format string `yaml:"format,omitempty"`

	// TimeZone sets the timezone for the date and time.
	// See [Time.LoadLocation](https://golang.org/pkg/time/#Time.LoadLocation).
	// Defaults to the local timezone.
	TimeZone string `yaml:"timezone,omitempty"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	// LocalBuild *beta* describes how to do a build on the local docker daemon
	// and optionally push to a repository.
	LocalBuild *LocalBuild `yaml:"local,omitempty" yamltags:"oneOf=build"`

	// GoogleCloudBuild *beta* describes how to do a remote build on
	// [Google Cloud Build](https://cloud.google.com/cloud-build/).
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild,omitempty" yamltags:"oneOf=build"`

	// KanikoBuild *beta* describes how to do an on-cluster build using
	// [Kaniko](https://github.com/GoogleContainerTools/kaniko).
	KanikoBuild *KanikoBuild `yaml:"kaniko,omitempty" yamltags:"oneOf=build"`
}

// LocalBuild *beta* describes how to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	// Push should images be pushed to a registry.
	// If not specified, images are pushed only if the current Kubernetes context
	// connects to a remote cluster.
	Push *bool `yaml:"push,omitempty"`

	// UseDockerCLI use `docker` command-line interface instead of Docker Engine APIs.
	UseDockerCLI bool `yaml:"useDockerCLI,omitempty"`

	// UseBuildkit use BuildKit to build Docker images. If unspecified, uses the Docker default.
	UseBuildkit *bool `yaml:"useBuildkit,omitempty"`
}

// GoogleCloudBuild *beta* describes how to do a remote build on
// [Google Cloud Build](https://cloud.google.com/cloud-build/docs/).
// Docker and Jib artifacts can be built on Cloud Build. The `projectId` needs
// to be provided and the currently logged in user should be given permissions to trigger
// new builds.
type GoogleCloudBuild struct {
	// ProjectID is the ID of your Cloud Platform Project.
	// If it is not provided, Skaffold will guess it from the image name.
	// For example, given the artifact image name `gcr.io/myproject/image`, Skaffold
	// will use the `myproject` GCP project.
	ProjectID string `yaml:"projectId,omitempty"`

	// DiskSizeGb is the disk size of the VM that runs the build.
	// See [Cloud Build Reference](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions).
	DiskSizeGb int64 `yaml:"diskSizeGb,omitempty"`

	// MachineType is the type of the VM that runs the build.
	// See [Cloud Build Reference](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#buildoptions).
	MachineType string `yaml:"machineType,omitempty"`

	// Timeout is the amount of time (in seconds) that this build should be allowed to run.
	// See [Cloud Build Reference](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#resource-build).
	Timeout string `yaml:"timeout,omitempty"`

	// DockerImage is the image that runs a Docker build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/docker`.
	DockerImage string `yaml:"dockerImage,omitempty"`

	// MavenImage is the image that runs a Maven build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/mvn`.
	MavenImage string `yaml:"mavenImage,omitempty"`

	// GradleImage is the image that runs a Gradle build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/gradle`.
	GradleImage string `yaml:"gradleImage,omitempty"`
}

// LocalDir configures how Kaniko mounts sources directly via an `emptyDir` volume.
type LocalDir struct{}

// KanikoBuildContext contains the different fields available to specify
// a Kaniko build context.
type KanikoBuildContext struct {
	// GCSBucket is the CGS bucket to which sources are uploaded.
	// Kaniko will need access to that bucket to download the sources.
	GCSBucket string `yaml:"gcsBucket,omitempty" yamltags:"oneOf=buildContext"`

	// LocalDir configures how Kaniko mounts sources directly via an `emptyDir` volume.
	LocalDir *LocalDir `yaml:"localDir,omitempty" yamltags:"oneOf=buildContext"`
}

// KanikoCache configures Kaniko caching. If a cache is specified, Kaniko will
// use a remote cache which will speed up builds.
type KanikoCache struct {
	// Repo is a remote repository to store cached layers. If none is specified, one will be
	// inferred from the image name. See [Kaniko Caching](https://github.com/GoogleContainerTools/kaniko#caching).
	Repo string `yaml:"repo,omitempty"`
}

// KanikoBuild *beta* describes how to do an on-cluster build using
// [Kaniko](https://github.com/GoogleContainerTools/kaniko).
type KanikoBuild struct {
	// BuildContext defines where Kaniko gets the sources from.
	BuildContext *KanikoBuildContext `yaml:"buildContext,omitempty"`

	// Cache configures Kaniko caching. If a cache is specified, Kaniko will
	// use a remote cache which will speed up builds.
	Cache *KanikoCache `yaml:"cache,omitempty"`

	// AdditionalFlags are additional flags to be passed to Kaniko command line.
	// See [Kaniko Additional Flags](https://github.com/GoogleContainerTools/kaniko#additional-flags).
	AdditionalFlags []string `yaml:"flags,omitempty"`

	// PullSecret is the path to the secret key file.
	// See [Kaniko Documentation](https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster).
	PullSecret string `yaml:"pullSecret,omitempty"`

	// PullSecretName is the name of the Kubernetes secret for pulling the files
	// from the build context and pushing the final image.
	// Defaults to `kaniko-secret`.
	PullSecretName string `yaml:"pullSecretName,omitempty"`

	// Namespace is the Kubernetes namespace.
	// Defaults to current namespace in Kubernetes configuration.
	Namespace string `yaml:"namespace,omitempty"`

	// Timeout is the amount of time (in seconds) that this build is allowed to run.
	// Defaults to 20 minutes (`20m`).
	Timeout string `yaml:"timeout,omitempty"`

	// Image is the Docker image used by the Kaniko pod.
	// Defaults to the latest released version of `gcr.io/kaniko-project/executor`.
	Image string `yaml:"image,omitempty"`

	// DockerConfig describes how to mount the local Docker configuration into the
	// Kaniko pod.
	DockerConfig *DockerConfig `yaml:"dockerConfig,omitempty"`
}

// DockerConfig contains information about the docker `config.json` to mount.
type DockerConfig struct {
	// Path is the path to the docker `config.json`.
	Path string `yaml:"path,omitempty"`

	// SecretName is the Kubernetes secret that will hold the Docker configuration.
	SecretName string `yaml:"secretName,omitempty"`
}

// TestCase is a list of structure tests to run on images that Skaffold builds.
type TestCase struct {
	// ImageName is the artifact on which to run those tests.
	// For example: `gcr.io/k8s-skaffold/example`.
	ImageName string `yaml:"image" yamltags:"required"`

	// StructureTests lists the [Container Structure Tests](https://github.com/GoogleContainerTools/container-structure-test)
	// to run on that artifact.
	// For example: `["./test/*"]`.
	StructureTests []string `yaml:"structureTests,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps.
type DeployConfig struct {
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	// HelmDeploy *beta* uses the `helm` CLI to apply the charts to the cluster.
	HelmDeploy *HelmDeploy `yaml:"helm,omitempty" yamltags:"oneOf=deploy"`

	// KubectlDeploy *beta* uses a client side `kubectl apply` to deploy manifests.
	// You'll need a `kubectl` CLI version installed that's compatible with your cluster.
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty" yamltags:"oneOf=deploy"`

	// KustomizeDeploy *beta* uses the `kustomize` CLI to "patch" a deployment for a target environment.
	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize,omitempty" yamltags:"oneOf=deploy"`
}

// KubectlDeploy *beta* uses a client side `kubectl apply` to deploy manifests.
// You'll need a `kubectl` CLI version installed that's compatible with your cluster.
type KubectlDeploy struct {
	// Manifests lists the Kubernetes yaml or json manifests.
	// Defaults to `["k8s/*.yaml"]`.
	Manifests []string `yaml:"manifests,omitempty"`

	// RemoteManifests lists Kubernetes manifests in remote clusters.
	RemoteManifests []string `yaml:"remoteManifests,omitempty"`

	// Flags are additional flags passed to `kubectl`.
	Flags KubectlFlags `yaml:"flags,omitempty"`
}

// KubectlFlags are additional flags passed on the command
// line to kubectl either on every command (Global), on creations (Apply)
// or deletions (Delete).
type KubectlFlags struct {
	// Global are additional flags passed on every command.
	Global []string `yaml:"global,omitempty"`

	// Apply are additional flags passed on creations (`kubectl apply`).
	Apply []string `yaml:"apply,omitempty"`

	// Delete are additional flags passed on deletions (`kubectl delete`).
	Delete []string `yaml:"delete,omitempty"`
}

// HelmDeploy *beta* uses the `helm` CLI to apply the charts to the cluster.
type HelmDeploy struct {
	// Releases is a list of Helm releases.
	Releases []HelmRelease `yaml:"releases,omitempty" yamltags:"required"`

	// Flags are additional option flags that are passed on the command
	// line to `helm`.
	Flags HelmDeployFlags `yaml:"flags,omitempty"`
}

// HelmDeployFlags are additional option flags that are passed on the command
// line to `helm`.
type HelmDeployFlags struct {
	// Global are additional flags passed on every command.
	Global []string `yaml:"global,omitempty"`

	// Install are additional flags passed to (`helm install`).
	Install []string `yaml:"install,omitempty"`

	// Upgrade are additional flags passed to (`helm upgrade`).
	Upgrade []string `yaml:"upgrade,omitempty"`
}

// KustomizeDeploy *beta* uses the `kustomize` CLI to "patch" a deployment for a target environment.
type KustomizeDeploy struct {
	// KustomizePath is the path to Kustomization files.
	// Defaults to `.`.
	KustomizePath string `yaml:"path,omitempty"`

	// Flags are additional flags passed to `kubectl`.
	Flags KubectlFlags `yaml:"flags,omitempty"`
}

// HelmRelease describes a helm release to be deployed.
type HelmRelease struct {
	// Name is the name of the Helm release.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// ChartPath is the path to the Helm chart.
	ChartPath string `yaml:"chartPath,omitempty" yamltags:"required"`

	// ValuesFiles are the paths to the Helm `values` files.
	ValuesFiles []string `yaml:"valuesFiles,omitempty"`

	// Values are key-value pairs supplementing the Helm `values` file.
	Values map[string]string `yaml:"values,omitempty,omitempty"`

	// Namespace is the Kubernetes namespace.
	Namespace string `yaml:"namespace,omitempty"`

	// Version is the version of the chart.
	Version string `yaml:"version,omitempty"`

	// SetValues are key-value pairs.
	// If present, Skaffold will send `--set` flag to Helm CLI and append all pairs after the flag.
	SetValues map[string]string `yaml:"setValues,omitempty"`

	// SetValueTemplates are key-value pairs.
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

	// SkipBuildDependencies should build dependencies be skipped.
	SkipBuildDependencies bool `yaml:"skipBuildDependencies,omitempty"`

	// Overrides are key-value pairs.
	// If present, Skaffold will build a Helm `values` file that overrides
	// the original and use it to call Helm CLI (`--f` flag).
	Overrides util.HelmOverrides `yaml:"overrides,omitempty"`

	// Packaged parameters for packaging helm chart (`helm package`).
	Packaged *HelmPackaged `yaml:"packaged,omitempty"`

	// ImageStrategy adds image configurations to the Helm `values` file.
	ImageStrategy HelmImageStrategy `yaml:"imageStrategy,omitempty"`
}

// HelmPackaged parameters for packaging helm chart (`helm package`).
type HelmPackaged struct {
	// Version sets the `version` on the chart to this semver version.
	Version string `yaml:"version,omitempty"`

	// AppVersion sets the `appVersion` on the chart to this version.
	AppVersion string `yaml:"appVersion,omitempty"`
}

// HelmImageStrategy adds image configurations to the Helm `values` file.
type HelmImageStrategy struct {
	HelmImageConfig `yaml:",inline"`
}

// HelmImageConfig describes an image configuration.
type HelmImageConfig struct {
	// HelmFQNConfig is the image configuration uses the syntax `IMAGE-NAME=IMAGE-REPOSITORY:IMAGE-TAG`.
	HelmFQNConfig *HelmFQNConfig `yaml:"fqn,omitempty" yamltags:"oneOf=helmImageStrategy"`

	// HelmConventionConfig is the image configuration uses the syntax `IMAGE-NAME.repository=IMAGE-REPOSITORY, IMAGE-NAME.tag=IMAGE-TAG`.
	HelmConventionConfig *HelmConventionConfig `yaml:"helm,omitempty" yamltags:"oneOf=helmImageStrategy"`
}

// HelmFQNConfig is the image config to use the FullyQualifiedImageName as param to set.
type HelmFQNConfig struct {
	// Property defines the image config.
	Property string `yaml:"property,omitempty"`
}

// HelmConventionConfig is the image config in the syntax of image.repository and image.tag.
type HelmConventionConfig struct {
}

// Artifact are the items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	// ImageName is the name of the image to be built.
	// For example: `gcr.io/k8s-skaffold/example`.
	ImageName string `yaml:"image,omitempty" yamltags:"required"`

	// Workspace is the directory containing the artifact's sources.
	// Defaults to `.`.
	Workspace string `yaml:"context,omitempty"`

	// Sync *alpha* lists local files synced to pods instead
	// of triggering an image build when modified.
	// This is a mapping of local files to sync to remote folders.
	// For example: `{"*.py": ".", "css/**/*.css": "app/css"}`.
	Sync map[string]string `yaml:"sync,omitempty"`

	// ArtifactType describes how to build an artifact.
	ArtifactType `yaml:",inline"`

	// BuilderPlugin is the plugin used to build this artifact.
	BuilderPlugin *BuilderPlugin `yaml:"plugin,omitempty"`
}

// Profile *beta* profiles are used to override any `build`, `test` or `deploy` configuration.
type Profile struct {
	// Name is a unique profile name.
	// For example: `profile-prod`.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// Build replaces the main `build` configuration.
	Build BuildConfig `yaml:"build,omitempty"`

	// Test replaces the main `test` configuration.
	Test []*TestCase `yaml:"test,omitempty"`

	// Deploy replaces the main `deploy` configuration.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// Patches lists patches applied to the configuration.
	// Patches use the JSON patch notation.
	Patches []JSONPatch `yaml:"patches,omitempty"`

	// Activation criteria by which a profile can be auto-activated.
	Activation []Activation `yaml:"activation,omitempty"`
}

// JSONPatch patch to be applied by a profile.
type JSONPatch struct {
	// Op is the operation carried by the patch: `add`, `remove`, `replace`, `move`, `copy` or `test`.
	// Defaults to `replace`.
	Op string `yaml:"op,omitempty"`

	// Path is the position in the yaml where the operation takes place.
	// For example, this targets the `dockerfile` of the first artifact built.
	// For example: `/build/artifacts/0/docker/dockerfile`.
	Path string `yaml:"path,omitempty" yamltags:"required"`

	// From is the source position in the yaml, used for `copy` or `move` operations.
	From string `yaml:"from,omitempty"`

	// Value is the value to apply. Can be any portion of yaml.
	Value *util.YamlpatchNode `yaml:"value,omitempty"`
}

// Activation criteria by which a profile is auto-activated.
type Activation struct {
	// Env is a `key=value` pair. The profile is auto-activated if an Environment
	// Variable `key` has value `value`.
	// For example: `ENV=production`.
	Env string `yaml:"env,omitempty"`

	// KubeContext is a Kubernetes context for which the profile is auto-activated.
	// For example: `minikube`.
	KubeContext string `yaml:"kubeContext,omitempty"`

	// Command is a Skaffold command for which the profile is auto-activated.
	// For example: `dev`.
	Command string `yaml:"command,omitempty"`
}

// ArtifactType describes how to build an artifact.
type ArtifactType struct {
	// DockerArtifact *beta* describes an artifact built from a Dockerfile.
	DockerArtifact *DockerArtifact `yaml:"docker,omitempty" yamltags:"oneOf=artifact"`

	// BazelArtifact *beta* requires bazel CLI to be installed and the sources to
	// contain [Bazel](https://bazel.build/) configuration files.
	BazelArtifact *BazelArtifact `yaml:"bazel,omitempty" yamltags:"oneOf=artifact"`

	// JibMavenArtifact *alpha* builds images using the
	// [Jib plugin for Maven](https://github.com/GoogleContainerTools/jib/tree/master/jib-maven-plugin).
	JibMavenArtifact *JibMavenArtifact `yaml:"jibMaven,omitempty" yamltags:"oneOf=artifact"`

	// JibGradleArtifact *alpha* builds images using the
	// [Jib plugin for Gradle](https://github.com/GoogleContainerTools/jib/tree/master/jib-gradle-plugin).
	JibGradleArtifact *JibGradleArtifact `yaml:"jibGradle,omitempty" yamltags:"oneOf=artifact"`
}

// DockerArtifact *beta* describes an artifact built from a Dockerfile,
// usually using `docker build`.
type DockerArtifact struct {
	// DockerfilePath locates the Dockerfile relative to workspace.
	// Defaults to `Dockerfile`.
	DockerfilePath string `yaml:"dockerfile,omitempty"`

	// Target is the Dockerfile target name to build.
	Target string `yaml:"target,omitempty"`

	// BuildArgs are arguments passed to the docker build.
	// For example: `{"key1": "value1", "key2": "value2"}`.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`

	// CacheFrom lists the Docker images used as cache sources.
	// For example: `["golang:1.10.1-alpine3.7", "alpine:3.7"]`.
	CacheFrom []string `yaml:"cacheFrom,omitempty"`
}

// BazelArtifact *beta* describes an artifact built with [Bazel](https://bazel.build/).
type BazelArtifact struct {
	// BuildTarget is the `bazel build` target to run.
	// For example: `//:skaffold_example.tar`.
	BuildTarget string `yaml:"target,omitempty" yamltags:"required"`

	// BuildArgs are additional args to pass to `bazel build`.
	// For example: `["-flag", "--otherflag"]`.
	BuildArgs []string `yaml:"args,omitempty"`
}

// JibMavenArtifact *alpha* builds images using the
// [Jib plugin for Maven](https://github.com/GoogleContainerTools/jib/tree/master/jib-maven-plugin).
type JibMavenArtifact struct {
	// Module selects which Maven module to build, for a multi module project.
	Module string `yaml:"module"`

	// Profile selects which Maven profile to activate.
	Profile string `yaml:"profile"`

	// Flags are additional build flags passed to Maven.
	// For example: `["-x", "-DskipTests"]`.
	Flags []string `yaml:"args,omitempty"`
}

// JibGradleArtifact *alpha* builds images using the
// [Jib plugin for Gradle](https://github.com/GoogleContainerTools/jib/tree/master/jib-gradle-plugin).
type JibGradleArtifact struct {
	// Project selects which Gradle project to build.
	Project string `yaml:"project"`

	// Flags are additional build flags passed to Gradle.
	// For example: `["--no-build-cache"]`.
	Flags []string `yaml:"args,omitempty"`
}
