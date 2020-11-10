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

package v2beta9

import (
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// !!! WARNING !!! This config version is already released, please DO NOT MODIFY the structs in this file.
const Version string = "skaffold/v2beta9"

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

	// Pipeline defines the Build/Test/Deploy phases.
	Pipeline `yaml:",inline"`

	// Profiles *beta* can override be used to `build`, `test` or `deploy` configuration.
	Profiles []Profile `yaml:"profiles,omitempty"`
}

// Metadata holds an optional name of the project.
type Metadata struct {
	// Name is an identifier for the project.
	Name string `yaml:"name,omitempty"`
}

// Pipeline describes a Skaffold pipeline.
type Pipeline struct {
	// Build describes how images are built.
	Build BuildConfig `yaml:"build,omitempty"`

	// Test describes how images are tested.
	Test []*TestCase `yaml:"test,omitempty"`

	// Deploy describes how images are deployed.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// PortForward describes user defined resources to port-forward.
	PortForward []*PortForwardResource `yaml:"portForward,omitempty"`
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// ResourceType describes the Kubernetes resource types used for port forwarding.
type ResourceType string

// PortForwardResource describes a resource to port forward.
type PortForwardResource struct {
	// Type is the Kubernetes type that should be port forwarded.
	// Acceptable resource types include: `Service`, `Pod` and Controller resource type that has a pod spec: `ReplicaSet`, `ReplicationController`, `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`.
	Type ResourceType `yaml:"resourceType,omitempty"`

	// Name is the name of the Kubernetes resource to port forward.
	Name string `yaml:"resourceName,omitempty"`

	// Namespace is the namespace of the resource to port forward.
	Namespace string `yaml:"namespace,omitempty"`

	// Port is the resource port that will be forwarded.
	Port int `yaml:"port,omitempty"`

	// Address is the local address to bind to. Defaults to the loopback address 127.0.0.1.
	Address string `yaml:"address,omitempty"`

	// LocalPort is the local port to forward to. If the port is unavailable, Skaffold will choose a random open port to forward to. *Optional*.
	LocalPort int `yaml:"localPort,omitempty"`
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
}

// ShaTagger *beta* tags images with their sha256 digest.
type ShaTagger struct{}

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

	// UseBuildkit use BuildKit to build Docker images.
	UseBuildkit bool `yaml:"useBuildkit,omitempty"`

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

	// Concurrency is how many artifacts can be built concurrently. 0 means "no-limit".
	// Defaults to `0`.
	Concurrency int `yaml:"concurrency,omitempty"`

	// WorkerPool configures a pool of workers to run the build.
	WorkerPool string `yaml:"workerPool,omitempty"`
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
	//TTL Cache timeout in hours.
	TTL string `yaml:"ttl,omitempty"`
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

	// Annotations describes the Kubernetes annotations for the pod.
	Annotations map[string]string `yaml:"annotations,omitempty"`

	// RunAsUser defines the UID to request for running the container.
	// If omitted, no SeurityContext will be specified for the pod and will therefore be inherited
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

	// StatusCheckDeadlineSeconds *beta* is the deadline for deployments to stabilize in seconds.
	StatusCheckDeadlineSeconds int `yaml:"statusCheckDeadlineSeconds,omitempty"`

	// KubeContext is the Kubernetes context that Skaffold should deploy to.
	// For example: `minikube`.
	KubeContext string `yaml:"kubeContext,omitempty"`

	// Logs configures how container logs are printed as a result of a deployment.
	Logs LogsConfig `yaml:"logs,omitempty"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. All three deployer types can be used at the same
// time for hybrid workflows.
type DeployType struct {
	// HelmDeploy *beta* uses the `helm` CLI to apply the charts to the cluster.
	HelmDeploy *HelmDeploy `yaml:"helm,omitempty"`

	// KptDeploy *alpha* uses the `kpt` CLI to manage and deploy manifests.
	KptDeploy *KptDeploy `yaml:"kpt,omitempty"`

	// KubectlDeploy *beta* uses a client side `kubectl apply` to deploy manifests.
	// You'll need a `kubectl` CLI version installed that's compatible with your cluster.
	KubectlDeploy *KubectlDeploy `yaml:"kubectl,omitempty"`

	// KustomizeDeploy *beta* uses the `kustomize` CLI to "patch" a deployment for a target environment.
	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize,omitempty"`
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

	// DefaultNamespace is the default namespace passed to kubectl on deployment if no other override is given.
	DefaultNamespace *string `yaml:"defaultNamespace,omitempty"`
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
	// KustomizePaths is the path to Kustomization files.
	// Defaults to `["."]`.
	KustomizePaths []string `yaml:"paths,omitempty"`

	// Flags are additional flags passed to `kubectl`.
	Flags KubectlFlags `yaml:"flags,omitempty"`

	// BuildArgs are additional args passed to `kustomize build`.
	BuildArgs []string `yaml:"buildArgs,omitempty"`

	// DefaultNamespace is the default namespace passed to kubectl on deployment if no other override is given.
	DefaultNamespace *string `yaml:"defaultNamespace,omitempty"`
}

// KptDeploy *alpha* uses the `kpt` CLI to manage and deploy manifests.
type KptDeploy struct {
	// Dir is the path to the config directory (Required).
	// By default, the Dir contains the application configurations,
	// [kustomize config files](https://kubectl.docs.kubernetes.io/pages/examples/kustomize.html)
	// and [declarative kpt functions](https://googlecontainertools.github.io/kpt/guides/consumer/function/#declarative-run).
	Dir string `yaml:"dir" yamltags:"required"`

	// Fn adds additional configurations for `kpt fn`.
	Fn KptFn `yaml:"fn,omitempty"`

	// Live adds additional configurations for `kpt live`.
	Live KptLive `yaml:"live,omitempty"`
}

// KptFn adds additional configurations used when calling `kpt fn`.
type KptFn struct {
	// FnPath is the directory to discover the declarative kpt functions.
	// If not provided, kpt deployer uses `kpt.Dir`.
	FnPath string `yaml:"fnPath,omitempty"`

	// Image is a kpt function image to run the configs imperatively. If provided, kpt.fn.fnPath
	// will be ignored.
	Image string `yaml:"image,omitempty"`

	// NetworkName is the docker network name to run the kpt function containers (default "bridge").
	NetworkName string `yaml:"networkName,omitempty"`

	// GlobalScope sets the global scope for the kpt functions. see `kpt help fn run`.
	GlobalScope bool `yaml:"globalScope,omitempty"`

	// Network enables network access for the kpt function containers.
	Network bool `yaml:"network,omitempty"`

	// Mount is a list of storage options to mount to the fn image.
	Mount []string `yaml:"mount,omitempty"`

	// SinkDir is the directory to where the manipulated resource output is stored.
	SinkDir string `yaml:"sinkDir,omitempty"`
}

// KptLive adds additional configurations used when calling `kpt live`.
type KptLive struct {
	// Apply sets the kpt inventory directory.
	Apply KptApplyInventory `yaml:"apply,omitempty"`

	// Options adds additional configurations for `kpt live apply` commands.
	Options KptApplyOptions `yaml:"options,omitempty"`
}

// KptApplyInventory sets the kpt inventory directory.
type KptApplyInventory struct {
	// Dir is equivalent to the dir in `kpt live apply <dir>`. If not provided,
	// kpt deployer will create a hidden directory `.kpt-hydrated` to store the manipulated
	// resource output and the kpt inventory-template.yaml file.
	Dir string `yaml:"dir,omitempty"`

	// InventoryID *alpha* is the identifier for a group of applied resources.
	// This value is only needed when the `kpt live` is working on a pre-applied cluster resources.
	InventoryID string `yaml:"inventoryID,omitempty"`

	// InventoryNamespace *alpha* sets the inventory namespace.
	InventoryNamespace string `yaml:"inventoryNamespace,omitempty"`
}

// KptApplyOptions adds additional configurations used when calling `kpt live apply`.
type KptApplyOptions struct {
	// PollPeriod sets for the polling period for resource statuses. Default to 2s.
	PollPeriod string `yaml:"pollPeriod,omitempty"`

	// PrunePropagationPolicy sets the propagation policy for pruning.
	// Possible settings are Background, Foreground, Orphan.
	// Default to "Background".
	PrunePropagationPolicy string `yaml:"prunePropagationPolicy,omitempty"`

	// PruneTimeout sets the time threshold to wait for all pruned resources to be deleted.
	PruneTimeout string `yaml:"pruneTimeout,omitempty"`

	// ReconcileTimeout sets the time threshold to wait for all resources to reach the current status.
	ReconcileTimeout string `yaml:"reconcileTimeout,omitempty"`
}

// HelmRelease describes a helm release to be deployed.
type HelmRelease struct {
	// Name is the name of the Helm release.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// ChartPath is the path to the Helm chart.
	ChartPath string `yaml:"chartPath,omitempty" yamltags:"required"`

	// ValuesFiles are the paths to the Helm `values` files.
	ValuesFiles []string `yaml:"valuesFiles,omitempty"`

	// ArtifactOverrides are key value pairs where the
	// key represents the parameter used in the `--set-string` Helm CLI flag to define a container
	// image and the value corresponds to artifact i.e. `ImageName` defined in `Build.Artifacts` section.
	// The resulting command-line is controlled by `ImageStrategy`.
	ArtifactOverrides util.FlatMap `yaml:"artifactOverrides,omitempty"`

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
	SetFiles map[string]string `yaml:"setFiles,omitempty"`

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
	// Ignored when `remote: true`.
	SkipBuildDependencies bool `yaml:"skipBuildDependencies,omitempty"`

	// UseHelmSecrets instructs skaffold to use secrets plugin on deployment.
	UseHelmSecrets bool `yaml:"useHelmSecrets,omitempty"`

	// Remote specifies whether the chart path is remote, or exists on the host filesystem.
	Remote bool `yaml:"remote,omitempty"`

	// UpgradeOnChange specifies whether to upgrade helm chart on code changes.
	// Default is `true` when helm chart is local (`remote: false`).
	// Default is `false` if `remote: true`.
	UpgradeOnChange *bool `yaml:"upgradeOnChange,omitempty"`

	// Overrides are key-value pairs.
	// If present, Skaffold will build a Helm `values` file that overrides
	// the original and use it to call Helm CLI (`--f` flag).
	Overrides util.HelmOverrides `yaml:"overrides,omitempty"`

	// Packaged parameters for packaging helm chart (`helm package`).
	Packaged *HelmPackaged `yaml:"packaged,omitempty"`

	// ImageStrategy controls how an `ArtifactOverrides` entry is
	// turned into `--set-string` Helm CLI flag or flags.
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

	// Sync *beta* lists local files synced to pods instead
	// of triggering an image build when modified.
	// If no files are listed, sync all the files and infer the destination.
	// Defaults to `infer: ["**/*"]`.
	Sync *Sync `yaml:"sync,omitempty"`

	// ArtifactType describes how to build an artifact.
	ArtifactType `yaml:",inline"`

	// Dependencies describes build artifacts that this artifact depends on.
	Dependencies []*ArtifactDependency `yaml:"requires,omitempty"`
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
	Builder string `yaml:"builder" yamltags:"required"`

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

	// ProjectDescriptor is the path to the project descriptor file.
	// Defaults to `project.toml` if it exists.
	ProjectDescriptor string `yaml:"projectDescriptor,omitempty"`

	// Dependencies are the file dependencies that skaffold should watch for both rebuilding and file syncing for this artifact.
	Dependencies *BuildpackDependencies `yaml:"dependencies,omitempty"`
}

// BuildpackDependencies *alpha* is used to specify dependencies for an artifact built by buildpacks.
type BuildpackDependencies struct {
	// Paths should be set to the file dependencies for this artifact, so that the skaffold file watcher knows when to rebuild and perform file synchronization.
	Paths []string `yaml:"paths,omitempty" yamltags:"oneOf=dependency"`

	// Ignore specifies the paths that should be ignored by skaffold's file watcher. If a file exists in both `paths` and in `ignore`, it will be ignored, and will be excluded from both rebuilds and file synchronization.
	// Will only work in conjunction with `paths`.
	Ignore []string `yaml:"ignore,omitempty"`
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
	// For example: `{"key1": "value1", "key2": "value2"}`.
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`

	// NetworkMode is passed through to docker and overrides the
	// network configuration of docker builder. If unset, use whatever
	// is configured in the underlying docker daemon. Valid modes are
	// `host`: use the host's networking stack.
	// `bridge`: use the bridged network configuration.
	// `none`: no networking in the container.
	NetworkMode string `yaml:"network,omitempty"`

	// CacheFrom lists the Docker images used as cache sources.
	// For example: `["golang:1.10.1-alpine3.7", "alpine:3.7"]`.
	CacheFrom []string `yaml:"cacheFrom,omitempty"`

	// NoCache used to pass in --no-cache to docker build to prevent caching.
	NoCache bool `yaml:"noCache,omitempty"`

	// Secret contains information about a local secret passed to `docker build`,
	// along with optional destination information.
	Secret *DockerSecret `yaml:"secret,omitempty"`
}

// DockerSecret contains information about a local secret passed to `docker build`,
// along with optional destination information.
type DockerSecret struct {
	// ID is the id of the secret.
	ID string `yaml:"id,omitempty" yamltags:"required"`

	// Source is the path to the secret on the host machine.
	Source string `yaml:"src,omitempty"`

	// Destination is the path in the container to mount the secret.
	Destination string `yaml:"dst,omitempty"`
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
