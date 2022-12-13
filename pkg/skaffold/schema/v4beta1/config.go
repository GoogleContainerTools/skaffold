/*
Copyright 2021 The Skaffold Authors

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

package v4beta1

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// This config version is not yet released, it is SAFE TO MODIFY the structs in this file.
const Version string = "skaffold/v4beta1"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}

// SkaffoldConfig holds the fields parsed from the Skaffold configuration file (skaffold.yaml).
type SkaffoldConfig struct {
	// APIVersion is the version of the configuration.
	APIVersion string `yaml:"apiVersion" yamltags:"required"`

	// Kind is always `Config`. Defaults to `Config`.
	Kind string `yaml:"kind" yamltags:"required"`

	// Metadata holds additional information about the config.
	Metadata Metadata `yaml:"metadata,omitempty"`

	// Dependencies describes a list of other required configs for the current config.
	Dependencies []ConfigDependency `yaml:"requires,omitempty"`

	// Pipeline defines the Build/Test/Deploy phases.
	Pipeline `yaml:",inline"`

	// Profiles *beta* can override be used to `build`, `test` or `deploy` configuration.
	Profiles []Profile `yaml:"profiles,omitempty"`
}

// Metadata holds an optional name of the project.
type Metadata struct {
	// Name is an identifier for the project.
	Name string `yaml:"name,omitempty"`

	// Labels is a map of labels identifying the project.
	Labels map[string]string `yaml:"labels,omitempty"`

	// Annotations is a map of annotations providing additional
	// metadata about the project.
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// Pipeline describes a Skaffold pipeline.
type Pipeline struct {
	// Build describes how images are built.
	Build BuildConfig `yaml:"build,omitempty"`

	// Test describes how images are tested.
	Test []*TestCase `yaml:"test,omitempty"`

	// Render describes how the original manifests are hydrated, validated and transformed.
	Render RenderConfig `yaml:"manifests,omitempty"`

	// Deploy describes how the manifests are deployed.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// PortForward describes user defined resources to port-forward.
	PortForward []*PortForwardResource `yaml:"portForward,omitempty"`

	// ResourceSelector describes user defined filters describing how skaffold should treat objects/fields during rendering.
	ResourceSelector ResourceSelectorConfig `yaml:"resourceSelector,omitempty"`

	// Verify describes how images are verified (via verification tests).
	Verify []*VerifyTestCase `yaml:"verify,omitempty"`
}

// GitInfo contains information on the origin of skaffold configurations cloned from a git repository.
type GitInfo struct {
	// Repo is the git repository the package should be cloned from.  e.g. `https://github.com/GoogleContainerTools/skaffold.git`.
	Repo string `yaml:"repo" yamltags:"required"`

	// Path is the relative path from the repo root to the skaffold configuration file. eg. `getting-started/skaffold.yaml`.
	Path string `yaml:"path,omitempty"`

	// Ref is the git ref the package should be cloned from. eg. `master` or `main`.
	Ref string `yaml:"ref,omitempty"`

	// Sync when set to `true` will reset the cached repository to the latest commit from remote on every run. To use the cached repository with uncommitted changes or unpushed commits, it needs to be set to `false`.
	Sync *bool `yaml:"sync,omitempty"`
}

// ConfigDependency describes a dependency on another skaffold configuration.
type ConfigDependency struct {
	// Names includes specific named configs within the file path. If empty, then all configs in the file are included.
	Names []string `yaml:"configs,omitempty"`

	// Path describes the path to the file containing the required configs.
	Path string `yaml:"path,omitempty" skaffold:"filepath" yamltags:"oneOf=paths"`

	// GitRepo describes a remote git repository containing the required configs.
	GitRepo *GitInfo `yaml:"git,omitempty" yamltags:"oneOf=paths"`

	// ActiveProfiles describes the list of profiles to activate when resolving the required configs. These profiles must exist in the imported config.
	ActiveProfiles []ProfileDependency `yaml:"activeProfiles,omitempty"`
}

// ProfileDependency describes a mapping from referenced config profiles to the current config profiles.
// If the current config is activated with a profile in this mapping then the dependency configs are also activated with the corresponding mapped profiles.
type ProfileDependency struct {
	// Name describes name of the profile to activate in the dependency config. It should exist in the dependency config.
	Name string `yaml:"name" yamltags:"required"`

	// ActivatedBy describes a list of profiles in the current config that when activated will also activate the named profile in the dependency config. If empty then the named profile is always activated.
	ActivatedBy []string `yaml:"activatedBy,omitempty"`
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// ResourceType describes the Kubernetes resource types used for port forwarding.
type ResourceType string

// PortForwardResource describes a resource to port forward.
type PortForwardResource struct {
	// Type is the resource type that should be port forwarded.
	// Acceptable resource types include kubernetes types: `Service`, `Pod` and Controller resource type that has a pod spec: `ReplicaSet`, `ReplicationController`, `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`.
	// Standalone `Container` is also valid for Docker deployments.
	Type ResourceType `yaml:"resourceType,omitempty"`

	// Name is the name of the Kubernetes resource or local container to port forward.
	Name string `yaml:"resourceName,omitempty"`

	// Namespace is the namespace of the resource to port forward. Does not apply to local containers.
	Namespace string `yaml:"namespace,omitempty"`

	// Port is the resource port that will be forwarded.
	Port util.IntOrString `yaml:"port,omitempty"`

	// Address is the local address to bind to. Defaults to the loopback address 127.0.0.1.
	Address string `yaml:"address,omitempty"`

	// LocalPort is the local port to forward to. If the port is unavailable, Skaffold will choose a random open port to forward to. *Optional*.
	LocalPort int `yaml:"localPort,omitempty"`
}

// ResourceSelectorConfig contains all the configuration needed by the deploy steps.
type ResourceSelectorConfig struct {
	// Allow configures an allowlist for transforming manifests.
	Allow []ResourceFilter `yaml:"allow,omitempty"`
	// Deny configures an allowlist for transforming manifests.
	Deny []ResourceFilter `yaml:"deny,omitempty"`
}

// BuildConfig contains all the configuration for the build steps.
type BuildConfig struct {
	// Artifacts lists the images you're going to be building.
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`

	// InsecureRegistries is a list of registries declared by the user to be insecure.
	// These registries will be connected to via HTTP instead of HTTPS.
	InsecureRegistries []string `yaml:"insecureRegistries,omitempty"`

	// TagPolicy *beta* determines how images are tagged.
	// A few strategies are provided here, although you most likely won't need to care!
	// If not specified, it defaults to `gitCommit: {variant: Tags}`.
	TagPolicy TagPolicy `yaml:"tagPolicy,omitempty"`

	// Platforms is the list of platforms to build all artifact images for.
	// It can be overridden by the individual artifact's `platforms` property.
	// If the target builder cannot build for atleast one of the specified platforms, then the build fails.
	// Each platform is of the format `os[/arch[/variant]]`, e.g., `linux/amd64`.
	// Example: `["linux/amd64", "linux/arm64"]`.
	Platforms []string `yaml:"platforms,omitempty"`

	BuildType `yaml:",inline"`
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

	// CustomTemplateTagger *beta* tags images with a configurable template string *composed of other taggers*.
	CustomTemplateTagger *CustomTemplateTagger `yaml:"customTemplate,omitempty" yamltags:"oneOf=tag"`

	// InputDigest *beta* tags images with their sha256 digest of their content.
	InputDigest *InputDigest `yaml:"inputDigest,omitempty" yamltags:"oneOf=tag"`
}

// ShaTagger *beta* tags images with their sha256 digest.
type ShaTagger struct{}

// InputDigest *beta* tags hashes the image content.
type InputDigest struct{}

// GitTagger *beta* tags images with the git tag or commit of the artifact's workspace.
type GitTagger struct {
	// Variant determines the behavior of the git tagger. Valid variants are:
	// `Tags` (default): use git tags or fall back to abbreviated commit hash.
	// `CommitSha`: use the full git commit sha.
	// `AbbrevCommitSha`: use the abbreviated git commit sha.
	// `TreeSha`: use the full tree hash of the artifact workingdir.
	// `AbbrevTreeSha`: use the abbreviated tree hash of the artifact workingdir.
	Variant string `yaml:"variant,omitempty"`

	// Prefix adds a fixed prefix to the tag.
	Prefix string `yaml:"prefix,omitempty"`

	// IgnoreChanges specifies whether to omit the `-dirty` postfix if there are uncommitted changes.
	IgnoreChanges bool `yaml:"ignoreChanges,omitempty"`
}

// EnvTemplateTagger *beta* tags images with a configurable template string.
type EnvTemplateTagger struct {
	// Template used to produce the image name and tag.
	// See golang [text/template](https://golang.org/pkg/text/template/).
	// The template is executed against the current environment,
	// with those variables injected.
	// For example: `{{.RELEASE}}`.
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

// CustomTemplateTagger *beta* tags images with a configurable template string.
type CustomTemplateTagger struct {
	// Template used to produce the image name and tag.
	// See golang [text/template](https://golang.org/pkg/text/template/).
	// The template is executed against the provided components with those variables injected.
	// For example: `{{.DATE}}` where DATE references a TaggerComponent.
	Template string `yaml:"template,omitempty" yamltags:"required"`

	// Components lists TaggerComponents that the template (see field above) can be executed against.
	Components []TaggerComponent `yaml:"components,omitempty"`
}

// TaggerComponent *beta* is a component of CustomTemplateTagger.
type TaggerComponent struct {
	// Name is an identifier for the component.
	Name string `yaml:"name,omitempty"`

	// Component is a tagging strategy to be used in CustomTemplateTagger.
	Component TagPolicy `yaml:",inline" yamltags:"skipTrim"`
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

	// Cluster *beta* describes how to do an on-cluster build.
	Cluster *ClusterDetails `yaml:"cluster,omitempty" yamltags:"oneOf=build"`
}

// LocalBuild *beta* describes how to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	// Push should images be pushed to a registry.
	// If not specified, images are pushed only if the current Kubernetes context
	// connects to a remote cluster.
	Push *bool `yaml:"push,omitempty"`

	// TryImportMissing whether to attempt to import artifacts from
	// Docker (either a local or remote registry) if not in the cache.
	TryImportMissing bool `yaml:"tryImportMissing,omitempty"`

	// UseDockerCLI use `docker` command-line interface instead of Docker Engine APIs.
	UseDockerCLI bool `yaml:"useDockerCLI,omitempty"`

	// UseBuildkit use BuildKit to build Docker images. If unspecified, uses the Docker default.
	UseBuildkit *bool `yaml:"useBuildkit,omitempty"`

	// Concurrency is how many artifacts can be built concurrently. 0 means "no-limit".
	// Defaults to `1`.
	Concurrency *int `yaml:"concurrency,omitempty"`
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

	// Logging specifies the logging mode.
	// Valid modes are:
	// `LOGGING_UNSPECIFIED`: The service determines the logging mode.
	// `LEGACY`: Stackdriver logging and Cloud Storage logging are enabled (default).
	// `GCS_ONLY`: Only Cloud Storage logging is enabled.
	// See [Cloud Build Reference](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#loggingmode).
	Logging string `yaml:"logging,omitempty"`

	// LogStreamingOption specifies the behavior when writing build logs to Google Cloud Storage.
	// Valid options are:
	// `STREAM_DEFAULT`: Service may automatically determine build log streaming behavior.
	// `STREAM_ON`:  Build logs should be streamed to Google Cloud Storage.
	// `STREAM_OFF`: Build logs should not be streamed to Google Cloud Storage; they will be written when the build is completed.
	// See [Cloud Build Reference](https://cloud.google.com/cloud-build/docs/api/reference/rest/v1/projects.builds#logstreamingoption).
	LogStreamingOption string `yaml:"logStreamingOption,omitempty"`

	// DockerImage is the image that runs a Docker build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/docker`.
	DockerImage string `yaml:"dockerImage,omitempty"`

	// KanikoImage is the image that runs a Kaniko build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/kaniko-project/executor`.
	KanikoImage string `yaml:"kanikoImage,omitempty"`

	// MavenImage is the image that runs a Maven build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/mvn`.
	MavenImage string `yaml:"mavenImage,omitempty"`

	// GradleImage is the image that runs a Gradle build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/cloud-builders/gradle`.
	GradleImage string `yaml:"gradleImage,omitempty"`

	// PackImage is the image that runs a Cloud Native Buildpacks build.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/k8s-skaffold/pack`.
	PackImage string `yaml:"packImage,omitempty"`

	// KoImage is the image that runs a ko build.
	// The image must contain Skaffold, Go, and a shell (runnable as `sh`) that supports here documents.
	// See [Cloud Builders](https://cloud.google.com/cloud-build/docs/cloud-builders).
	// Defaults to `gcr.io/k8s-skaffold/skaffold`.
	KoImage string `yaml:"koImage,omitempty"`

	// Concurrency is how many artifacts can be built concurrently. 0 means "no-limit".
	// Defaults to `0`.
	Concurrency int `yaml:"concurrency,omitempty"`

	// WorkerPool configures a pool of workers to run the build.
	WorkerPool string `yaml:"workerPool,omitempty"`

	// Region configures the region to run the build. If WorkerPool is configured, the region will
	// be deduced from the WorkerPool configuration. If neither WorkerPool nor Region is configured,
	// the build will be run in global(non-regional).
	// See [Cloud Build locations](https://cloud.google.com/build/docs/locations).
	Region string `yaml:"region,omitempty"`

	// PlatformEmulatorInstallStep specifies a pre-build step to install the required tooling for QEMU emulation on the GoogleCloudBuild containers. This enables performing cross-platform builds on GoogleCloudBuild.
	// If unspecified, Skaffold uses the `docker/binfmt` image by default.
	PlatformEmulatorInstallStep *PlatformEmulatorInstallStep `yaml:"platformEmulatorInstallStep,omitempty"`

	// ServiceAccount is the Google Cloud platform service account used by Cloud Build.
	// If unspecified, it defaults to the Cloud Build service account generated when
	// the Cloud Build API is enabled.
	ServiceAccount string `yaml:"serviceAccount,omitempty"`
}

// PlatformEmulatorInstallStep specifies a pre-build step to install the required tooling for QEMU emulation on the GoogleCloudBuild containers. This enables performing cross-platform builds on GoogleCloudBuild.
type PlatformEmulatorInstallStep struct {
	// Image specifies the image that will install the required tooling for QEMU emulation on the GoogleCloudBuild containers.
	Image string `yaml:"image" yamltags:"required"`
	// Args specifies arguments passed to the emulator installer image.
	Args []string `yaml:"args,omitempty"`
	// Entrypoint specifies the ENTRYPOINT argument to the emulator installer image.
	Entrypoint string `yaml:"entrypoint,omitempty"`
}

// KanikoCache configures Kaniko caching. If a cache is specified, Kaniko will
// use a remote cache which will speed up builds.
type KanikoCache struct {
	// Repo is a remote repository to store cached layers. If none is specified, one will be
	// inferred from the image name. See [Kaniko Caching](https://github.com/GoogleContainerTools/kaniko#caching).
	Repo string `yaml:"repo,omitempty"`
	// HostPath specifies a path on the host that is mounted to each pod as read only cache volume containing base images.
	// If set, must exist on each node and prepopulated with kaniko-warmer.
	HostPath string `yaml:"hostPath,omitempty"`
	// TTL Cache timeout in hours.
	TTL string `yaml:"ttl,omitempty"`
	// CacheCopyLayers enables caching of copy layers.
	CacheCopyLayers bool `yaml:"cacheCopyLayers,omitempty"`
}

// ClusterDetails *beta* describes how to do an on-cluster build.
type ClusterDetails struct {
	// HTTPProxy for kaniko pod.
	HTTPProxy string `yaml:"HTTP_PROXY,omitempty"`

	// HTTPSProxy for kaniko pod.
	HTTPSProxy string `yaml:"HTTPS_PROXY,omitempty"`

	// PullSecretPath is the path to the Google Cloud service account secret key file.
	PullSecretPath string `yaml:"pullSecretPath,omitempty"`

	// PullSecretName is the name of the Kubernetes secret for pulling base images
	// and pushing the final image. If given, the secret needs to contain the Google Cloud
	// service account secret key under the key `kaniko-secret`.
	// Defaults to `kaniko-secret`.
	PullSecretName string `yaml:"pullSecretName,omitempty"`

	// PullSecretMountPath is the path the pull secret will be mounted at within the running container.
	PullSecretMountPath string `yaml:"pullSecretMountPath,omitempty"`

	// Namespace is the Kubernetes namespace.
	// Defaults to current namespace in Kubernetes configuration.
	Namespace string `yaml:"namespace,omitempty"`

	// Timeout is the amount of time (in seconds) that this build is allowed to run.
	// Defaults to 20 minutes (`20m`).
	Timeout string `yaml:"timeout,omitempty"`

	// DockerConfig describes how to mount the local Docker configuration into a pod.
	DockerConfig *DockerConfig `yaml:"dockerConfig,omitempty"`

	// ServiceAccountName describes the Kubernetes service account to use for the pod.
	// Defaults to 'default'.
	ServiceAccountName string `yaml:"serviceAccount,omitempty"`

	// Tolerations describes the Kubernetes tolerations for the pod.
	Tolerations []v1.Toleration `yaml:"tolerations,omitempty"`

	// NodeSelector describes the Kubernetes node selector for the pod.
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty"`

	// Annotations describes the Kubernetes annotations for the pod.
	Annotations map[string]string `yaml:"annotations,omitempty"`

	// RunAsUser defines the UID to request for running the container.
	// If omitted, no SecurityContext will be specified for the pod and will therefore be inherited
	// from the service account.
	RunAsUser *int64 `yaml:"runAsUser,omitempty"`

	// Resources define the resource requirements for the kaniko pod.
	Resources *ResourceRequirements `yaml:"resources,omitempty"`

	// Concurrency is how many artifacts can be built concurrently. 0 means "no-limit".
	// Defaults to `0`.
	Concurrency int `yaml:"concurrency,omitempty"`

	// Volumes defines container mounts for ConfigMap and Secret resources.
	Volumes []v1.Volume `yaml:"volumes,omitempty"`

	// RandomPullSecret adds a random UUID postfix to the default name of the pull secret to facilitate parallel builds, e.g. kaniko-secretdocker-cfgfd154022-c761-416f-8eb3-cf8258450b85.
	RandomPullSecret bool `yaml:"randomPullSecret,omitempty"`

	// RandomDockerConfigSecret adds a random UUID postfix to the default name of the docker secret to facilitate parallel builds, e.g. docker-cfgfd154022-c761-416f-8eb3-cf8258450b85.
	RandomDockerConfigSecret bool `yaml:"randomDockerConfigSecret,omitempty"`
}

// DockerConfig contains information about the docker `config.json` to mount.
type DockerConfig struct {
	// Path is the path to the docker `config.json`.
	Path string `yaml:"path,omitempty"`

	// SecretName is the Kubernetes secret that contains the `config.json` Docker configuration.
	// Note that the expected secret type is not 'kubernetes.io/dockerconfigjson' but 'Opaque'.
	SecretName string `yaml:"secretName,omitempty"`
}

// ResourceRequirements describes the resource requirements for the kaniko pod.
type ResourceRequirements struct {
	// Requests [resource requests](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container) for the Kaniko pod.
	Requests *ResourceRequirement `yaml:"requests,omitempty"`

	// Limits [resource limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container) for the Kaniko pod.
	Limits *ResourceRequirement `yaml:"limits,omitempty"`
}

// ResourceRequirement stores the CPU/Memory requirements for the pod.
type ResourceRequirement struct {
	// CPU the number cores to be used.
	// For example: `2`, `2.0` or `200m`.
	CPU string `yaml:"cpu,omitempty"`

	// Memory the amount of memory to allocate to the pod.
	// For example: `1Gi` or `1000Mi`.
	Memory string `yaml:"memory,omitempty"`

	// EphemeralStorage the amount of Ephemeral storage to allocate to the pod.
	// For example: `1Gi` or `1000Mi`.
	EphemeralStorage string `yaml:"ephemeralStorage,omitempty"`

	// ResourceStorage the amount of resource storage to allocate to the pod.
	// For example: `1Gi` or `1000Mi`.
	ResourceStorage string `yaml:"resourceStorage,omitempty"`
}

// TestCase is a list of tests to run on images that Skaffold builds.
type TestCase struct {
	// ImageName is the artifact on which to run those tests.
	// For example: `gcr.io/k8s-skaffold/example`.
	ImageName string `yaml:"image" yamltags:"required"`

	// Workspace is the directory containing the test sources.
	// Defaults to `.`.
	Workspace string `yaml:"context,omitempty" skaffold:"filepath"`

	// CustomTests lists the set of custom tests to run after an artifact is built.
	CustomTests []CustomTest `yaml:"custom,omitempty"`

	// StructureTests lists the [Container Structure Tests](https://github.com/GoogleContainerTools/container-structure-test)
	// to run on that artifact.
	// For example: `["./test/*"]`.
	StructureTests []string `yaml:"structureTests,omitempty" skaffold:"filepath"`

	// StructureTestArgs lists additional configuration arguments passed to `container-structure-test` binary.
	// For example: `["--driver=tar", "--no-color", "-q"]`.
	StructureTestArgs []string `yaml:"structureTestsArgs,omitempty"`
}

// VerifyTestCase is a list of tests to run on images that Skaffold builds.
type VerifyTestCase struct {
	// Name is the name descriptor for the verify test.
	Name string `yaml:"name" yamltags:"required"`
	// Container is the container information for the verify test.
	Container v1.Container `yaml:"container,omitempty" yamltags:"oneOf=verifyType"`
}

// RenderConfig contains all the configuration needed by the render steps.
type RenderConfig struct {

	// Generate defines the dry manifests from a variety of sources.
	Generate `yaml:",inline"`

	// Transform defines a set of transformation operations to run in series.
	Transform *[]Transformer `yaml:"transform,omitempty"`

	// Validate defines a set of validator operations to run in series.
	Validate *[]Validator `yaml:"validate,omitempty"`

	// Output is the path to the hydrated directory.
	Output string `yaml:"output,omitempty"`
}

// Generate defines the dry manifests from a variety of sources.
type Generate struct {
	// RawK8s defines the raw kubernetes resources.
	RawK8s []string `yaml:"rawYaml,omitempty" skaffold:"filepath"`

	// RemoteManifests lists Kubernetes manifests in remote clusters.
	RemoteManifests []RemoteManifest `yaml:"remoteManifests,omitempty"`

	// Kustomize defines the paths to be modified with kustomize, along with extra
	// flags to be passed to kustomize.
	Kustomize *Kustomize `yaml:"kustomize,omitempty"`

	// Helm defines the helm charts used in the application.
	// NOTE: Defines cherts in this section to render via helm but
	// deployed via kubectl or kpt deployer.
	// To use helm to deploy, please see deploy.helm section.
	Helm *Helm `yaml:"helm,omitempty"`

	// Kpt defines the kpt resources in the application.
	Kpt []string `yaml:"kpt,omitempty" skaffold:"filepath"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after every render.
	LifecycleHooks RenderHooks `yaml:"hooks,omitempty"`
}

// RemoteManifest defines the paths to be modified with kustomize, along with
// extra flags to be passed to kustomize.
type RemoteManifest struct {
	// Manifest specifies the Kubernetes manifest in the remote cluster.
	Manifest string `yaml:"manifest,omitempty"`

	// KubeContext is the Kubernetes context that Skaffold should deploy to.
	// For example: `minikube`.
	KubeContext string `yaml:"kubeContext,omitempty"`
}

// Kustomize defines the paths to be modified with kustomize, along with
// extra flags to be passed to kustomize.
type Kustomize struct {
	// Paths is the path to Kustomization files.
	// Defaults to `["."]`.
	Paths []string `yaml:"paths,omitempty" skaffold:"filepath"`

	// BuildArgs are additional args passed to `kustomize build`.
	BuildArgs []string `yaml:"buildArgs,omitempty"`
}

// Helm defines the manifests from helm releases.
type Helm struct {
	// Flags are additional option flags that are passed on the command
	// line to `helm`.
	Flags HelmDeployFlags `yaml:"flags,omitempty"`

	// Releases is a list of Helm releases.
	Releases []HelmRelease `yaml:"releases,omitempty" yamltags:"required"`
}

// Transformer describes the supported kpt transformers.
type Transformer struct {
	// Name is the transformer name. Can only accept skaffold whitelisted tools.
	Name string `yaml:"name" yamltags:"required"`
	// ConfigMap allows users to provide additional config data to the kpt function.
	ConfigMap []string `yaml:"configMap,omitempty"`
}

// Validator describes the supported kpt validators.
type Validator struct {
	// Name is the Validator name. Can only accept skaffold whitelisted tools.
	Name string `yaml:"name" yamltags:"required"`
	// ConfigMap allows users to provide additional config data to the kpt function.
	ConfigMap []string `yaml:"configMap,omitempty"`
}

// KptDeploy contains all the configuration needed by the deploy steps.
type KptDeploy struct {
	// Dir is equivalent to the dir in `kpt live apply <dir>`. If not provided, skaffold deploys from the default
	// hydrated path `<WORKDIR>/.kpt-pipeline`.
	Dir string `yaml:"dir,omitempty"`

	// ApplyFlags are additional flags passed to `kpt live apply`.
	ApplyFlags []string `yaml:"applyFlags,omitempty"`

	// Flags are kpt global flags.
	Flags []string `yaml:"flags,omitempty"`

	// Name *alpha* is the inventory object name.
	Name string `yaml:"name,omitempty"`

	// InventoryID *alpha* is the inventory ID which annotates the resources being lively applied by kpt.
	InventoryID string `yaml:"inventoryID,omitempty"`

	// InventoryNamespace *alpha* sets the inventory namespace.
	InventoryNamespace string `yaml:"namespace,omitempty"`

	// Force is used in `kpt live init`, which forces the inventory values to be updated, even if they are already set.
	Force bool `yaml:"false,omitempty"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after every deploy.
	LifecycleHooks DeployHooks `yaml:"-"`

	// DefaultNamespace is the default namespace passed to kpt on deployment if no other override is given.
	DefaultNamespace *string `yaml:"defaultNamespace,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps.
type DeployConfig struct {
	DeployType `yaml:",inline"`

	// StatusCheck *beta* enables waiting for deployments to stabilize.
	StatusCheck *bool `yaml:"statusCheck,omitempty"`

	// StatusCheckDeadlineSeconds *beta* is the deadline for deployments to stabilize in seconds.
	StatusCheckDeadlineSeconds int `yaml:"statusCheckDeadlineSeconds,omitempty"`

	// TolerateFailuresUntilDeadline configures the Skaffold "status-check" to tolerate failures
	// (flapping deployments, etc.) until the statusCheckDeadlineSeconds duration or k8s object
	// timeouts such as progressDeadlineSeconds, etc.
	TolerateFailuresUntilDeadline bool `yaml:"tolerateFailuresUntilDeadline,omitempty"`

	// KubeContext is the Kubernetes context that Skaffold should deploy to.
	// For example: `minikube`.
	KubeContext string `yaml:"kubeContext,omitempty"`

	// Logs configures how container logs are printed as a result of a deployment.
	Logs LogsConfig `yaml:"logs,omitempty"`

	// TransformableAllowList configures an allowlist for transforming manifests.
	TransformableAllowList []ResourceFilter `yaml:"-"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. All three deployer types can be used at the same
// time for hybrid workflows.
type DeployType struct {
	// DockerDeploy *alpha* uses the `docker` CLI to create application containers in Docker.
	DockerDeploy *DockerDeploy `yaml:"docker,omitempty"`

	// LegacyHelmDeploy *beta* uses the `helm` CLI to apply the charts to the cluster.
	LegacyHelmDeploy *LegacyHelmDeploy `yaml:"helm,omitempty"`

	// KptDeploy *alpha* uses the `kpt` CLI to manage and deploy manifests.
	KptDeploy *KptDeploy `yaml:"kpt,omitempty"`

	// KubectlDeploy *beta* uses a client side `kubectl apply` to deploy manifests.
	// You'll need a `kubectl` CLI version installed that's compatible with your cluster.
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty"`

	// CloudRunDeploy *alpha* deploys to Google Cloud Run using the Cloud Run v1 API.
	CloudRunDeploy *CloudRunDeploy `yaml:"cloudrun,omitempty"`
}

// CloudRunDeploy *alpha* deploys the container to Google Cloud Run.
type CloudRunDeploy struct {
	// ProjectID the GCP Project to use for Cloud Run.
	// If specified, all Services will be deployed to this project. If not specified,
	// each Service will be deployed to the project specified in `metadata.namespace` of
	// the Cloud Run manifest.
	ProjectID string `yaml:"projectid,omitempty"`

	// Region GCP location to use for the Cloud Run Deploy.
	// Must be one of the regions listed in https://cloud.google.com/run/docs/locations.
	Region string `yaml:"region,omitempty"`
}

// DockerDeploy uses the `docker` CLI to create application containers in Docker.
type DockerDeploy struct {
	// UseCompose tells skaffold whether or not to deploy using `docker-compose`.
	UseCompose bool `yaml:"useCompose,omitempty"`

	// Images are the container images to run in Docker.
	Images []string `yaml:"images" yamltags:"required"`
}

// KubectlDeploy *beta* uses a client side `kubectl apply` to deploy manifests.
// You'll need a `kubectl` CLI version installed that's compatible with your cluster.
type KubectlDeploy struct {
	// Flags are additional flags passed to `kubectl`.
	Flags KubectlFlags `yaml:"flags,omitempty"`

	// RemoteManifests lists Kubernetes manifests in remote clusters.
	RemoteManifests []string `yaml:"remoteManifests,omitempty"`

	// DefaultNamespace is the default namespace passed to kubectl on deployment if no other override is given.
	DefaultNamespace *string `yaml:"defaultNamespace,omitempty"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after every deploy.
	LifecycleHooks DeployHooks `yaml:"hooks,omitempty"`
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

	// DisableValidation passes the `--validate=false` flag to supported
	// `kubectl` commands when enabled.
	DisableValidation bool `yaml:"disableValidation,omitempty"`
}

// LegacyHelmDeploy *beta* uses the `helm` CLI to apply the charts to the cluster.
type LegacyHelmDeploy struct {
	// Releases is a list of Helm releases.
	Releases []HelmRelease `yaml:"releases,omitempty"`

	// Flags are additional option flags that are passed on the command
	// line to `helm`.
	Flags HelmDeployFlags `yaml:"flags,omitempty"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after every deploy.
	LifecycleHooks DeployHooks `yaml:"hooks,omitempty"`
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

// HelmRelease describes a helm release to be deployed.
type HelmRelease struct {
	// Name is the name of the Helm release.
	// It accepts environment variables via the go template syntax.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// ChartPath is the local path to a packaged Helm chart or an unpacked Helm chart directory.
	ChartPath string `yaml:"chartPath,omitempty" yamltags:"oneOf=chartSource" skaffold:"filepath"`

	// RemoteChart refers to a remote Helm chart reference or URL.
	RemoteChart string `yaml:"remoteChart,omitempty" yamltags:"oneOf=chartSource"`

	// ValuesFiles are the paths to the Helm `values` files.
	ValuesFiles []string `yaml:"valuesFiles,omitempty" skaffold:"filepath"`

	// Namespace is the Kubernetes namespace.
	Namespace string `yaml:"namespace,omitempty"`

	// Version is the version of the chart.
	Version string `yaml:"version,omitempty"`

	// SetValues are key-value pairs.
	// If present, Skaffold will send `--set` flag to Helm CLI and append all pairs after the flag.
	SetValues util.FlatMap `yaml:"setValues,omitempty"`

	// SetValueTemplates are key-value pairs.
	// If present, Skaffold will try to parse the value part of each key-value pair using
	// environment variables in the system, then send `--set` flag to Helm CLI and append
	// all parsed pairs after the flag.
	SetValueTemplates util.FlatMap `yaml:"setValueTemplates,omitempty"`

	// SetFiles are key-value pairs.
	// If present, Skaffold will send `--set-file` flag to Helm CLI and append all pairs after the flag.
	SetFiles map[string]string `yaml:"setFiles,omitempty" skaffold:"filepath"`

	// CreateNamespace if `true`, Skaffold will send `--create-namespace` flag to Helm CLI.
	// `--create-namespace` flag is available in Helm since version 3.2.
	// Defaults is `false`.
	CreateNamespace *bool `yaml:"createNamespace,omitempty"`

	// Wait if `true`, Skaffold will send `--wait` flag to Helm CLI.
	// Defaults to `false`.
	Wait bool `yaml:"wait,omitempty"`

	// RecreatePods if `true`, Skaffold will send `--recreate-pods` flag to Helm CLI
	// when upgrading a new version of a chart in subsequent dev loop deploy.
	// Defaults to `false`.
	RecreatePods bool `yaml:"recreatePods,omitempty"`

	// SkipBuildDependencies should build dependencies be skipped.
	// Ignored for `remoteChart`.
	SkipBuildDependencies bool `yaml:"skipBuildDependencies,omitempty"`

	// SkipTests should ignore helm test during manifests generation.
	// Defaults to `false`
	SkipTests bool `yaml:"skipTests,omitempty"`

	// UseHelmSecrets instructs skaffold to use secrets plugin on deployment.
	UseHelmSecrets bool `yaml:"useHelmSecrets,omitempty"`

	// Repo specifies the helm repository for remote charts.
	// If present, Skaffold will send `--repo` Helm CLI flag or flags.
	Repo string `yaml:"repo,omitempty"`

	// UpgradeOnChange specifies whether to upgrade helm chart on code changes.
	// Default is `true` when helm chart is local (has `chartPath`).
	// Default is `false` when helm chart is remote (has `remoteChart`).
	UpgradeOnChange *bool `yaml:"upgradeOnChange,omitempty"`

	// Overrides are key-value pairs.
	// If present, Skaffold will build a Helm `values` file that overrides
	// the original and use it to call Helm CLI (`--f` flag).
	Overrides util.HelmOverrides `yaml:"overrides,omitempty"`

	// Packaged parameters for packaging helm chart (`helm package`).
	Packaged *HelmPackaged `yaml:"packaged,omitempty"`
}

// HelmPackaged parameters for packaging helm chart (`helm package`).
type HelmPackaged struct {
	// Version sets the `version` on the chart to this semver version.
	Version string `yaml:"version,omitempty"`

	// AppVersion sets the `appVersion` on the chart to this version.
	AppVersion string `yaml:"appVersion,omitempty"`
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
	// ExplicitRegistry separates `image.registry` to the image config syntax. Useful for some charts e.g. `postgresql`.
	ExplicitRegistry bool `yaml:"explicitRegistry,omitempty"`
}

// LogsConfig configures how container logs are printed as a result of a deployment.
type LogsConfig struct {
	// Prefix defines the prefix shown on each log line. Valid values are
	// `container`: prefix logs lines with the name of the container.
	// `podAndContainer`: prefix logs lines with the names of the pod and of the container.
	// `auto`: same as `podAndContainer` except that the pod name is skipped if it's the same as the container name.
	// `none`: don't add a prefix.
	// Defaults to `auto`.
	Prefix string `yaml:"prefix,omitempty"`

	// JSONParse defines the rules for parsing/outputting json logs.
	JSONParse JSONParseConfig `yaml:"jsonParse,omitempty"`
}

// JSONParseConfig defines the rules for parsing/outputting json logs.
type JSONParseConfig struct {
	// Fields specifies which top level fields should be printed.
	Fields []string `yaml:"fields,omitempty"`
}

// Artifact are the items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	// ImageName is the name of the image to be built.
	// For example: `gcr.io/k8s-skaffold/example`.
	ImageName string `yaml:"image,omitempty" yamltags:"required"`

	// Workspace is the directory containing the artifact's sources.
	// Defaults to `.`.
	Workspace string `yaml:"context,omitempty" skaffold:"filepath"`

	// Sync *beta* lists local files synced to pods instead
	// of triggering an image build when modified.
	// If no files are listed, sync all the files and infer the destination.
	// Defaults to `infer: ["**/*"]`.
	Sync *Sync `yaml:"sync,omitempty"`

	// ArtifactType describes how to build an artifact.
	ArtifactType `yaml:",inline"`

	// Dependencies describes build artifacts that this artifact depends on.
	Dependencies []*ArtifactDependency `yaml:"requires,omitempty"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after each build of the target artifact.
	LifecycleHooks BuildHooks `yaml:"hooks,omitempty"`

	// Platforms is the list of platforms to build this artifact image for.
	// It overrides the values inferred through heuristics or provided in the top level `platforms` property or in the global config.
	// If the target builder cannot build for atleast one of the specified platforms, then the build fails.
	// Each platform is of the format `os[/arch[/variant]]`, e.g., `linux/amd64`.
	// Example: `["linux/amd64", "linux/arm64"]`.
	Platforms []string `yaml:"platforms,omitempty"`
}

// Sync *beta* specifies what files to sync into the container.
// This is a list of sync rules indicating the intent to sync for source files.
// If no files are listed, sync all the files and infer the destination.
// Defaults to `infer: ["**/*"]`.
type Sync struct {
	// Manual lists manual sync rules indicating the source and destination.
	Manual []*SyncRule `yaml:"manual,omitempty" yamltags:"oneOf=sync"`

	// Infer lists file patterns which may be synced into the container
	// The container destination is inferred by the builder
	// based on the instructions of a Dockerfile.
	// Available for docker and kaniko artifacts and custom
	// artifacts that declare dependencies on a dockerfile.
	Infer []string `yaml:"infer,omitempty" yamltags:"oneOf=sync"`

	// Auto delegates discovery of sync rules to the build system.
	// Only available for jib and buildpacks.
	Auto *bool `yaml:"auto,omitempty" yamltags:"oneOf=sync"`

	// LifecycleHooks describes a set of lifecycle hooks that are executed before and after each file sync action on the target artifact's containers.
	LifecycleHooks SyncHooks `yaml:"hooks,omitempty"`
}

// SyncRule specifies which local files to sync to remote folders.
type SyncRule struct {
	// Src is a glob pattern to match local paths against.
	// Directories should be delimited by `/` on all platforms.
	// For example: `"css/**/*.css"`.
	Src string `yaml:"src,omitempty" yamltags:"required"`

	// Dest is the destination path in the container where the files should be synced to.
	// For example: `"app/"`
	Dest string `yaml:"dest,omitempty" yamltags:"required"`

	// Strip specifies the path prefix to remove from the source path when
	// transplanting the files into the destination folder.
	// For example: `"css/"`
	Strip string `yaml:"strip,omitempty"`
}

// Profile is used to override any `build`, `test` or `deploy` configuration.
type Profile struct {
	// Name is a unique profile name.
	// For example: `profile-prod`.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// Activation criteria by which a profile can be auto-activated.
	// The profile is auto-activated if any one of the activations are triggered.
	// An activation is triggered if all of the criteria (env, kubeContext, command) are triggered.
	Activation []Activation `yaml:"activation,omitempty"`

	// RequiresAllActivations is the activation strategy of the profile.
	// When true, the profile is auto-activated only when all of its activations are triggered.
	// When false, the profile is auto-activated when any one of its activations is triggered.
	RequiresAllActivations bool `yaml:"requiresAllActivations,omitempty"`

	// Patches lists patches applied to the configuration.
	// Patches use the JSON patch notation.
	Patches []JSONPatch `yaml:"patches,omitempty"`

	// Pipeline contains the definitions to replace the default skaffold pipeline.
	Pipeline `yaml:",inline"`
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
	// Env is a `key=pattern` pair. The profile is auto-activated if an Environment
	// Variable `key` matches the pattern. If the pattern starts with `!`, activation
	// happens if the remaining pattern is _not_ matched. The pattern matches if the
	// Environment Variable value is exactly `pattern`, or the regex `pattern` is
	// found in it. An empty `pattern` (e.g. `env: "key="`) always only matches if
	// the Environment Variable is undefined or empty.
	// For example: `ENV=production`
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

	// KoArtifact builds images using [ko](https://github.com/google/ko).
	KoArtifact *KoArtifact `yaml:"ko,omitempty" yamltags:"oneOf=artifact"`

	// JibArtifact builds images using the
	// [Jib plugins for Maven or Gradle](https://github.com/GoogleContainerTools/jib/).
	JibArtifact *JibArtifact `yaml:"jib,omitempty" yamltags:"oneOf=artifact"`

	// KanikoArtifact builds images using [kaniko](https://github.com/GoogleContainerTools/kaniko).
	KanikoArtifact *KanikoArtifact `yaml:"kaniko,omitempty" yamltags:"oneOf=artifact"`

	// BuildpackArtifact builds images using [Cloud Native Buildpacks](https://buildpacks.io/).
	BuildpackArtifact *BuildpackArtifact `yaml:"buildpacks,omitempty" yamltags:"oneOf=artifact"`

	// CustomArtifact *beta* builds images using a custom build script written by the user.
	CustomArtifact *CustomArtifact `yaml:"custom,omitempty" yamltags:"oneOf=artifact"`
}

// ArtifactDependency describes a specific build dependency for an artifact.
type ArtifactDependency struct {
	// ImageName is a reference to an artifact's image name.
	ImageName string `yaml:"image" yamltags:"required"`
	// Alias is a token that is replaced with the image reference in the builder definition files.
	// For example, the `docker` builder will use the alias as a build-arg key.
	// Defaults to the value of `image`.
	Alias string `yaml:"alias,omitempty"`
}

// BuildpackArtifact *alpha* describes an artifact built using [Cloud Native Buildpacks](https://buildpacks.io/).
// It can be used to build images out of project's sources without any additional configuration.
type BuildpackArtifact struct {
	// Builder is the builder image used.
	Builder string `yaml:"builder,omitempty"`

	// RunImage overrides the stack's default run image.
	RunImage string `yaml:"runImage,omitempty"`

	// Env are environment variables, in the `key=value` form,  passed to the build.
	// Values can use the go template syntax.
	// For example: `["key1=value1", "key2=value2", "key3={{.ENV_VARIABLE}}"]`.
	Env []string `yaml:"env,omitempty"`

	// Buildpacks is a list of strings, where each string is a specific buildpack to use with the builder.
	// If you specify buildpacks the builder image automatic detection will be ignored. These buildpacks will be used to build the Image from your source code.
	// Order matters.
	Buildpacks []string `yaml:"buildpacks,omitempty"`

	// TrustBuilder indicates that the builder should be trusted.
	TrustBuilder bool `yaml:"trustBuilder,omitempty"`

	// ClearCache removes old cache volume associated with the specific image
	// and supplies a clean cache volume for build.
	ClearCache bool `yaml:"clearCache,omitempty"`

	// ProjectDescriptor is the path to the project descriptor file.
	// Defaults to `project.toml` if it exists.
	ProjectDescriptor string `yaml:"projectDescriptor,omitempty"`

	// Dependencies are the file dependencies that skaffold should watch for both rebuilding and file syncing for this artifact.
	Dependencies *BuildpackDependencies `yaml:"dependencies,omitempty"`

	// Volumes support mounting host volumes into the container.
	Volumes []*BuildpackVolume `yaml:"volumes,omitempty"`
}

// BuildpackDependencies *alpha* is used to specify dependencies for an artifact built by buildpacks.
type BuildpackDependencies struct {
	// Paths should be set to the file dependencies for this artifact, so that the skaffold file watcher knows when to rebuild and perform file synchronization.
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by skaffold's file watcher. If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	// Will only work in conjunction with `paths`.
	Ignore []string `yaml:"ignore,omitempty"`
}

// BuildpackVolume *alpha* is used to mount host volumes or directories in the build container.
type BuildpackVolume struct {
	// Host is the local volume or absolute directory of the path to mount.
	Host string `yaml:"host" skaffold:"filepath" yamltags:"required"`

	// Target is the path where the file or directory is available in the container.
	// It is strongly recommended to not specify locations under `/cnb` or `/layers`.
	Target string `yaml:"target" yamltags:"required"`

	// Options specify a list of comma-separated mount options.
	// Valid options are:
	// `ro` (default): volume contents are read-only.
	// `rw`: volume contents are readable and writable.
	// `volume-opt=<key>=<value>`: can be specified more than once, takes a key-value pair.
	Options string `yaml:"options,omitempty"`
}

// CustomArtifact *beta* describes an artifact built from a custom build script
// written by the user. It can be used to build images with builders that aren't directly integrated with skaffold.
type CustomArtifact struct {
	// BuildCommand is the command executed to build the image.
	BuildCommand string `yaml:"buildCommand,omitempty"`
	// Dependencies are the file dependencies that skaffold should watch for both rebuilding and file syncing for this artifact.
	Dependencies *CustomDependencies `yaml:"dependencies,omitempty"`
}

// CustomDependencies *beta* is used to specify dependencies for an artifact built by a custom build script.
// Either `dockerfile` or `paths` should be specified for file watching to work as expected.
type CustomDependencies struct {
	// Dockerfile should be set if the artifact is built from a Dockerfile, from which skaffold can determine dependencies.
	Dockerfile *DockerfileDependency `yaml:"dockerfile,omitempty" yamltags:"oneOf=dependency"`

	// Command represents a custom command that skaffold executes to obtain dependencies. The output of this command *must* be a valid JSON array.
	Command string `yaml:"command,omitempty" yamltags:"oneOf=dependency"`

	// Paths should be set to the file dependencies for this artifact, so that the skaffold file watcher knows when to rebuild and perform file synchronization.
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by skaffold's file watcher. If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	// Will only work in conjunction with `paths`.
	Ignore []string `yaml:"ignore,omitempty"`
}

// CustomTest describes the custom test command provided by the user.
// Custom tests are run after an image build whenever build or test dependencies are changed.
type CustomTest struct {
	// Command is the custom command to be executed.  If the command exits with a non-zero return
	// code, the test will be considered to have failed.
	Command string `yaml:"command" yamltags:"required"`

	// TimeoutSeconds sets the wait time for skaffold for the command to complete.
	// If unset or 0, Skaffold will wait until the command completes.
	TimeoutSeconds int `yaml:"timeoutSeconds,omitempty"`

	// Dependencies are additional test-specific file dependencies; changes to these files will re-run this test.
	Dependencies *CustomTestDependencies `yaml:"dependencies,omitempty"`
}

// CustomTestDependencies is used to specify dependencies for custom test command.
// `paths` should be specified for file watching to work as expected.
type CustomTestDependencies struct {
	// Command represents a command that skaffold executes to obtain dependencies. The output of this command *must* be a valid JSON array.
	Command string `yaml:"command,omitempty" yamltags:"oneOf=dependency"`

	// Paths locates the file dependencies for the command relative to workspace.
	// Paths should be set to the file dependencies for this command, so that the skaffold file watcher knows when to retest and perform file synchronization.
	// For example: `["src/test/**"]`
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by skaffold's file watcher. If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both retest and file synchronization.
	// Will only work in conjunction with `paths`.
	Ignore []string `yaml:"ignore,omitempty"`
}

// DockerfileDependency *beta* is used to specify a custom build artifact that is built from a Dockerfile. This allows skaffold to determine dependencies from the Dockerfile.
type DockerfileDependency struct {
	// Path locates the Dockerfile relative to workspace.
	Path string `yaml:"path,omitempty"`

	// BuildArgs are key/value pairs used to resolve values of `ARG` instructions in a Dockerfile.
	// Values can be constants or environment variables via the go template syntax.
	// For example: `{"key1": "value1", "key2": "value2", "key3": "'{{.ENV_VARIABLE}}'"}`.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`
}

// KanikoArtifact describes an artifact built from a Dockerfile,
// with kaniko.
type KanikoArtifact struct {

	// Cleanup to clean the filesystem at the end of the build.
	Cleanup bool `yaml:"cleanup,omitempty"`

	// Insecure if you want to push images to a plain HTTP registry.
	Insecure bool `yaml:"insecure,omitempty"`

	// InsecurePull if you want to pull images from a plain HTTP registry.
	InsecurePull bool `yaml:"insecurePull,omitempty"`

	// NoPush if you only want to build the image, without pushing to a registry.
	NoPush bool `yaml:"noPush,omitempty"`

	// Force building outside of a container.
	Force bool `yaml:"force,omitempty"`

	// LogTimestamp to add timestamps to log format.
	LogTimestamp bool `yaml:"logTimestamp,omitempty"`

	// Reproducible is used to strip timestamps out of the built image.
	Reproducible bool `yaml:"reproducible,omitempty"`

	// SingleSnapshot is takes a single snapshot of the filesystem at the end of the build.
	// So only one layer will be appended to the base image.
	SingleSnapshot bool `yaml:"singleSnapshot,omitempty"`

	// SkipTLS skips TLS certificate validation when pushing to a registry.
	SkipTLS bool `yaml:"skipTLS,omitempty"`

	// SkipTLSVerifyPull skips TLS certificate validation when pulling from a registry.
	SkipTLSVerifyPull bool `yaml:"skipTLSVerifyPull,omitempty"`

	// SkipUnusedStages builds only used stages if defined to true.
	// Otherwise it builds by default all stages, even the unnecessaries ones until it reaches the target stage / end of Dockerfile.
	SkipUnusedStages bool `yaml:"skipUnusedStages,omitempty"`

	// UseNewRun to Use the experimental run implementation for detecting changes without requiring file system snapshots.
	// In some cases, this may improve build performance by 75%.
	UseNewRun bool `yaml:"useNewRun,omitempty"`

	// WhitelistVarRun is used to ignore `/var/run` when taking image snapshot.
	// Set it to false to preserve /var/run/* in destination image.
	WhitelistVarRun bool `yaml:"whitelistVarRun,omitempty"`

	// DockerfilePath locates the Dockerfile relative to workspace.
	// Defaults to `Dockerfile`.
	DockerfilePath string `yaml:"dockerfile,omitempty"`

	// Target is to indicate which build stage is the target build stage.
	Target string `yaml:"target,omitempty"`

	// InitImage is the image used to run init container which mounts kaniko context.
	InitImage string `yaml:"initImage,omitempty"`

	// Image is the Docker image used by the Kaniko pod.
	// Defaults to the latest released version of `gcr.io/kaniko-project/executor`.
	Image string `yaml:"image,omitempty"`

	// DigestFile to specify a file in the container. This file will receive the digest of a built image.
	// This can be used to automatically track the exact image built by kaniko.
	DigestFile string `yaml:"digestFile,omitempty"`

	// ImageFSExtractRetry is the number of retries that should happen for extracting an image filesystem.
	ImageFSExtractRetry string `yaml:"imageFSExtractRetry,omitempty"`

	// ImageNameWithDigestFile specify a file to save the image name with digest of the built image to.
	ImageNameWithDigestFile string `yaml:"imageNameWithDigestFile,omitempty"`

	// LogFormat <text|color|json> to set the log format.
	LogFormat string `yaml:"logFormat,omitempty"`

	// OCILayoutPath is to specify a directory in the container where the OCI image layout of a built image will be placed.
	// This can be used to automatically track the exact image built by kaniko.
	OCILayoutPath string `yaml:"ociLayoutPath,omitempty"`

	// RegistryMirror if you want to use a registry mirror instead of default `index.docker.io`.
	RegistryMirror string `yaml:"registryMirror,omitempty"`

	// SnapshotMode is how Kaniko will snapshot the filesystem.
	SnapshotMode string `yaml:"snapshotMode,omitempty"`

	// PushRetry Set this flag to the number of retries that should happen for the push of an image to a remote destination.
	PushRetry string `yaml:"pushRetry,omitempty"`

	// TarPath is path to save the image as a tarball at path instead of pushing the image.
	TarPath string `yaml:"tarPath,omitempty"`

	// Verbosity <panic|fatal|error|warn|info|debug|trace> to set the logging level.
	Verbosity string `yaml:"verbosity,omitempty"`

	// InsecureRegistry is to use plain HTTP requests when accessing a registry.
	InsecureRegistry []string `yaml:"insecureRegistry,omitempty"`

	// SkipTLSVerifyRegistry skips TLS certificate validation when accessing a registry.
	SkipTLSVerifyRegistry []string `yaml:"skipTLSVerifyRegistry,omitempty"`

	// Env are environment variables passed to the kaniko pod.
	// It also accepts environment variables via the go template syntax.
	// For example: `[{"name": "key1", "value": "value1"}, {"name": "key2", "value": "value2"}, {"name": "key3", "value": "'{{.ENV_VARIABLE}}'"}]`.
	Env []v1.EnvVar `yaml:"env,omitempty"`

	// Cache configures Kaniko caching. If a cache is specified, Kaniko will
	// use a remote cache which will speed up builds.
	Cache *KanikoCache `yaml:"cache,omitempty"`

	// RegistryCertificate is to provide a certificate for TLS communication with a given registry.
	// my.registry.url: /path/to/the/certificate.cert is the expected format.
	RegistryCertificate map[string]*string `yaml:"registryCertificate,omitempty"`

	// Label key: value to set some metadata to the final image.
	// This is equivalent as using the LABEL within the Dockerfile.
	Label map[string]*string `yaml:"label,omitempty"`

	// BuildArgs are arguments passed to the docker build.
	// It also accepts environment variables and generated values via the go template syntax.
	// Exposed generated values: IMAGE_REPO, IMAGE_NAME, IMAGE_TAG.
	// For example: `{"key1": "value1", "key2": "value2", "key3": "'{{.ENV_VARIABLE}}'"}`.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`

	// VolumeMounts are volume mounts passed to kaniko pod.
	VolumeMounts []v1.VolumeMount `yaml:"volumeMounts,omitempty"`

	// ContextSubPath is to specify a sub path within the context.
	ContextSubPath string `yaml:"contextSubPath,omitempty" skaffold:"filepath"`
}

// DockerArtifact describes an artifact built from a Dockerfile,
// usually using `docker build`.
type DockerArtifact struct {
	// DockerfilePath locates the Dockerfile relative to workspace.
	// Defaults to `Dockerfile`.
	DockerfilePath string `yaml:"dockerfile,omitempty"`

	// Target is the Dockerfile target name to build.
	Target string `yaml:"target,omitempty"`

	// BuildArgs are arguments passed to the docker build.
	// For example: `{"key1": "value1", "key2": "{{ .ENV_VAR }}"}`.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`

	// NetworkMode is passed through to docker and overrides the
	// network configuration of docker builder. If unset, use whatever
	// is configured in the underlying docker daemon. Valid modes are
	// `host`: use the host's networking stack.
	// `bridge`: use the bridged network configuration.
	// `container:<name|id>`: reuse another container's network stack.
	// `none`: no networking in the container.
	NetworkMode string `yaml:"network,omitempty"`

	// AddHost lists add host.
	// For example: `["host1:ip1", "host2:ip2"]`.
	AddHost []string `yaml:"addHost,omitempty"`

	// CacheFrom lists the Docker images used as cache sources.
	// For example: `["golang:1.10.1-alpine3.7", "alpine:3.7"]`.
	CacheFrom []string `yaml:"cacheFrom,omitempty"`

	// CliFlags are any additional flags to pass to the local daemon during a build.
	// These flags are only used during a build through the Docker CLI.
	CliFlags []string `yaml:"cliFlags,omitempty"`

	// PullParent is used to attempt pulling the parent image even if an older image exists locally.
	PullParent bool `yaml:"pullParent,omitempty"`

	// NoCache set to true to pass in --no-cache to docker build, which will prevent caching.
	NoCache bool `yaml:"noCache,omitempty"`

	// Squash is used to pass in --squash to docker build to squash docker image layers into single layer.
	Squash bool `yaml:"squash,omitempty"`

	// Secrets is used to pass in --secret to docker build, `useBuildKit: true` is required.
	Secrets []*DockerSecret `yaml:"secrets,omitempty"`

	// SSH is used to pass in --ssh to docker build to use SSH agent. Format is "default|<id>[=<socket>|<key>[,<key>]]".
	SSH string `yaml:"ssh,omitempty"`
}

// DockerSecret is used to pass in --secret to docker build, `useBuildKit: true` is required.
type DockerSecret struct {
	// ID is the id of the secret.
	ID string `yaml:"id,omitempty" yamltags:"required"`

	// Source is the path to the secret on the host machine.
	Source string `yaml:"src,omitempty" yamltags:"oneOf=secretSource"`

	// Env is the environment variable name containing the secret value.
	Env string `yaml:"env,omitempty" yamltags:"oneOf=secretSource"`
}

// BazelArtifact describes an artifact built with [Bazel](https://bazel.build/).
type BazelArtifact struct {
	// BuildTarget is the `bazel build` target to run.
	// For example: `//:skaffold_example.tar`.
	BuildTarget string `yaml:"target,omitempty" yamltags:"required"`

	// BuildArgs are additional args to pass to `bazel build`.
	// For example: `["-flag", "--otherflag"]`.
	BuildArgs []string `yaml:"args,omitempty"`
}

// KoArtifact builds images using [ko](https://github.com/google/ko).
type KoArtifact struct {
	// BaseImage overrides the default ko base image (`gcr.io/distroless/static:nonroot`).
	// Corresponds to, and overrides, the `defaultBaseImage` in `.ko.yaml`.
	BaseImage string `yaml:"fromImage,omitempty"`

	// Dependencies are the file dependencies that Skaffold should watch for both rebuilding and file syncing for this artifact.
	Dependencies *KoDependencies `yaml:"dependencies,omitempty"`

	// Dir is the directory where the `go` tool will be run.
	// The value is a directory path relative to the `context` directory.
	// If empty, the `go` tool will run in the `context` directory.
	// Example: `./my-app-sources`.
	Dir string `yaml:"dir,omitempty"`

	// Env are environment variables, in the `key=value` form, passed to the build.
	// These environment variables are only used at build time.
	// They are _not_ set in the resulting container image.
	// For example: `["GOPRIVATE=git.example.com", "GOCACHE=/workspace/.gocache"]`.
	Env []string `yaml:"env,omitempty"`

	// Flags are additional build flags passed to `go build`.
	// For example: `["-trimpath", "-v"]`.
	Flags []string `yaml:"flags,omitempty"`

	// Labels are key-value string pairs to add to the image config.
	// For example: `{"foo":"bar"}`.
	Labels map[string]string `yaml:"labels,omitempty"`

	// Ldflags are linker flags passed to the builder.
	// For example: `["-buildid=", "-s", "-w"]`.
	Ldflags []string `yaml:"ldflags,omitempty"`

	// Main is the location of the main package. It is the pattern passed to `go build`.
	// If main is specified as a relative path, it is relative to the `context` directory.
	// If main is empty, the ko builder uses a default value of `.`.
	// If main is a pattern with wildcards, such as `./...`, the expansion must contain only one main package, otherwise ko fails.
	// Main is ignored if the `ImageName` starts with `ko://`.
	// Example: `./cmd/foo`.
	Main string `yaml:"main,omitempty"`
}

// KoDependencies is used to specify dependencies for an artifact built by ko.
type KoDependencies struct {
	// Paths should be set to the file dependencies for this artifact, so that the Skaffold file watcher knows when to rebuild and perform file synchronization.
	// Defaults to `["**/*.go"]`.
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by Skaffold's file watcher.
	// If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	Ignore []string `yaml:"ignore,omitempty"`
}

// JibArtifact builds images using the
// [Jib plugins for Maven and Gradle](https://github.com/GoogleContainerTools/jib/).
type JibArtifact struct {
	// Project selects which sub-project to build for multi-module builds.
	Project string `yaml:"project,omitempty"`

	// Flags are additional build flags passed to the builder.
	// For example: `["--no-build-cache"]`.
	Flags []string `yaml:"args,omitempty"`

	// Type the Jib builder type; normally determined automatically. Valid types are
	// `maven`: for Maven.
	// `gradle`: for Gradle.
	Type string `yaml:"type,omitempty"`

	// BaseImage overrides the configured jib base image.
	BaseImage string `yaml:"fromImage,omitempty"`
}

// BuildHooks describes the list of lifecycle hooks to execute before and after each artifact build step.
type BuildHooks struct {
	// PreHooks describes the list of lifecycle hooks to execute *before* each artifact build step.
	PreHooks []HostHook `yaml:"before,omitempty"`
	// PostHooks describes the list of lifecycle hooks to execute *after* each artifact build step.
	PostHooks []HostHook `yaml:"after,omitempty"`
}

// SyncHookItem describes a single lifecycle hook to execute before or after each artifact sync step.
type SyncHookItem struct {
	// HostHook describes a single lifecycle hook to run on the host machine.
	HostHook *HostHook `yaml:"host,omitempty" yamltags:"oneOf=sync_hook"`
	// ContainerHook describes a single lifecycle hook to run on a container.
	ContainerHook *ContainerHook `yaml:"container,omitempty" yamltags:"oneOf=sync_hook"`
}

// SyncHooks describes the list of lifecycle hooks to execute before and after each artifact sync step.
type SyncHooks struct {
	// PreHooks describes the list of lifecycle hooks to execute *before* each artifact sync step.
	PreHooks []SyncHookItem `yaml:"before,omitempty"`
	// PostHooks describes the list of lifecycle hooks to execute *after* each artifact sync step.
	PostHooks []SyncHookItem `yaml:"after,omitempty"`
}

// RenderHookItem describes a single lifecycle hook to execute before or after each deployer step.
type RenderHookItem struct {
	// HostHook describes a single lifecycle hook to run on the host machine.
	HostHook *HostHook `yaml:"host,omitempty" yamltags:"oneOf=render_hook"`
}

// RenderHooks describes the list of lifecycle hooks to execute before and after each render step.
type RenderHooks struct {
	// PreHooks describes the list of lifecycle hooks to execute *before* each render step. Container hooks will only run if the container exists from a previous deployment step (for instance the successive iterations of a dev-loop during `skaffold dev`).
	PreHooks []RenderHookItem `yaml:"before,omitempty"`
	// PostHooks describes the list of lifecycle hooks to execute *after* each render step.
	PostHooks []RenderHookItem `yaml:"after,omitempty"`
}

// DeployHookItem describes a single lifecycle hook to execute before or after each deployer step.
type DeployHookItem struct {
	// HostHook describes a single lifecycle hook to run on the host machine.
	HostHook *HostHook `yaml:"host,omitempty" yamltags:"oneOf=deploy_hook"`
	// ContainerHook describes a single lifecycle hook to run on a container.
	ContainerHook *NamedContainerHook `yaml:"container,omitempty" yamltags:"oneOf=deploy_hook"`
}

// DeployHooks describes the list of lifecycle hooks to execute before and after each deployer step.
type DeployHooks struct {
	// PreHooks describes the list of lifecycle hooks to execute *before* each deployer step. Container hooks will only run if the container exists from a previous deployment step (for instance the successive iterations of a dev-loop during `skaffold dev`).
	PreHooks []DeployHookItem `yaml:"before,omitempty"`
	// PostHooks describes the list of lifecycle hooks to execute *after* each deployer step.
	PostHooks []DeployHookItem `yaml:"after,omitempty"`
}

// HostHook describes a lifecycle hook definition to execute on the host machine.
type HostHook struct {
	// Command is the command to execute.
	Command []string `yaml:"command" yamltags:"required"`
	// OS is an optional slice of operating system names. If the host machine OS is different, then it skips execution.
	OS []string `yaml:"os,omitempty"`
	// Dir specifies the working directory of the command.
	// If empty, the command runs in the calling process's current directory.
	Dir string `yaml:"dir,omitempty" skaffold:"filepath"`
}

// ContainerHook describes a lifecycle hook definition to execute on a container. The container name is inferred from the scope in which this hook is defined.
type ContainerHook struct {
	// Command is the command to execute.
	Command []string `yaml:"command" yamltags:"required"`
}

// NamedContainerHook describes a lifecycle hook definition to execute on a named container.
type NamedContainerHook struct {
	// ContainerHook describes a lifecycle hook definition to execute on a container.
	ContainerHook `yaml:",inline" yamlTags:"skipTrim"`
	// PodName is the name of the pod to execute the command in.
	PodName string `yaml:"podName" yamltags:"required"`
	// ContainerName is the name of the container to execute the command in.
	ContainerName string `yaml:"containerName,omitempty"`
}

// ResourceFilter contains definition to filter which resource to transform.
type ResourceFilter struct {
	// GroupKind is the compact format of a resource type.
	GroupKind string `yaml:"groupKind" yamltags:"required"`
	// Image is an optional slice of JSON-path-like paths of where to rewrite images.
	Image []string `yaml:"image,omitempty"`
	// Labels is an optional slice of JSON-path-like paths of where to add a labels block if missing.
	Labels []string `yaml:"labels,omitempty"`
	// PodSpec is an optional slice of JSON-path-like paths of where pod spec properties can be overwritten.
	PodSpec []string `yaml:"podSpec,omitempty"`
}

// UnmarshalYAML provides a custom unmarshaller to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func (clusterDetails *ClusterDetails) UnmarshalYAML(value *yaml.Node) error {
	// We do this as follows
	// 1. We zero out the fields in the node that require custom processing
	// 2. We unmarshal all the non special fields using the aliased type resource
	//    we use an alias type to avoid recursion caused by invoking this function infinitely
	// 3. We deserialize the special fields as required.
	type ClusterDetailsForUnmarshaling ClusterDetails

	volumes, remaining, err := util.UnmarshalClusterVolumes(value)

	if err != nil {
		return err
	}

	// Unmarshal the remaining values
	aux := (*ClusterDetailsForUnmarshaling)(clusterDetails)
	err = yaml.Unmarshal(remaining, aux)

	if err != nil {
		return err
	}

	clusterDetails.Volumes = volumes
	return nil
}

// UnmarshalYAML provides a custom unmarshaller to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func (ka *KanikoArtifact) UnmarshalYAML(value *yaml.Node) error {
	// We do this as follows
	// 1. We zero out the fields in the node that require custom processing
	// 2. We unmarshal all the non special fields using the aliased type resource
	//    we use an alias type to avoid recursion caused by invoking this function infinitely
	// 3. We deserialize the special fields as required.
	type KanikoArtifactForUnmarshaling KanikoArtifact

	mounts, remaining, err := util.UnmarshalKanikoArtifact(value)

	if err != nil {
		return err
	}

	// Unmarshal the remaining values
	aux := (*KanikoArtifactForUnmarshaling)(ka)
	err = yaml.Unmarshal(remaining, aux)

	if err != nil {
		return err
	}

	ka.VolumeMounts = mounts
	return nil
}

// MarshalYAML provides a custom marshaller to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func (clusterDetails *ClusterDetails) MarshalYAML() (interface{}, error) {
	// We do this as follows
	// 1. We zero out the fields in the node that require custom processing
	// 2. We marshall all the non special fields using the aliased type resource
	//    we use an alias type to avoid recursion caused by invoking this function infinitely
	// 3. We unmarshal to a map
	// 4. We marshal the special fields to json and unmarshal to a map
	//    * This leverages the json struct annotations to marshal as expected
	// 5. We combine the two maps and return
	type ClusterDetailsForUnmarshaling ClusterDetails

	// Marshal volumes to a list. Use json because the Kubernetes resources have json annotations.
	volumes := clusterDetails.Volumes

	j, err := json.Marshal(volumes)

	if err != nil {
		return err, nil
	}

	vList := []interface{}{}

	if err := json.Unmarshal(j, &vList); err != nil {
		return nil, err
	}

	// Make a deep copy of clusterDetails because we need to zero out volumes and we don't want to modify the
	// current object.
	aux := &ClusterDetailsForUnmarshaling{}

	b, err := json.Marshal(clusterDetails)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return nil, err
	}

	aux.Volumes = nil

	marshaled, err := yaml.Marshal(aux)

	if err != nil {
		return nil, err
	}

	m := map[string]interface{}{}

	err = yaml.Unmarshal(marshaled, m)

	if len(vList) > 0 {
		m["volumes"] = vList
	}
	return m, err
}

// MarshalYAML provides a custom marshaller to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func (ka *KanikoArtifact) MarshalYAML() (interface{}, error) {
	// We do this as follows
	// 1. We zero out the fields in the node that require custom processing
	// 2. We marshal all the non special fields using the aliased type resource
	//    we use an alias type to avoid recursion caused by invoking this function infinitely
	// 3. We unmarshal to a map
	// 4. We marshal the special fields to json and unmarshal to a map
	//    * This leverages the json struct annotations to marshal as expected
	// 5. We combine the two maps and return
	type KanikoArtifactForUnmarshaling KanikoArtifact

	// Marshal volumes to a map. User json because the Kubernetes resources have json annotations.
	volumeMounts := ka.VolumeMounts

	j, err := json.Marshal(volumeMounts)

	if err != nil {
		return err, nil
	}

	vList := []interface{}{}

	if err := json.Unmarshal(j, &vList); err != nil {
		return nil, err
	}

	// Make a deep copy of kanikoArtifact because we need to zero out volumeMounts and we don't want to modify the
	// current object.
	aux := &KanikoArtifactForUnmarshaling{}

	b, err := json.Marshal(ka)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return nil, err
	}
	aux.VolumeMounts = nil

	marshaled, err := yaml.Marshal(aux)

	if err != nil {
		return nil, err
	}

	m := map[string]interface{}{}

	err = yaml.Unmarshal(marshaled, m)

	if len(vList) > 0 {
		m["volumeMounts"] = vList
	}
	return m, err
}
