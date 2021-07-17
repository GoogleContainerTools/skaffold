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

package inspect

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Options holds flag values for the various `skaffold inspect` commands
type Options struct {
	// Filename is the `skaffold.yaml` file path
	Filename string
	// RepoCacheDir is the directory for the remote git repository cache
	RepoCacheDir string
	// OutFormat is the output format. One of: json
	OutFormat string
	// Modules is the module filter for specific commands
	Modules []string
	// Strict specifies the error-tolerance for specific commands
	Strict bool

	ModulesOptions
	ProfilesOptions
	BuildEnvOptions
}

// ModulesOptions holds flag values for various `skaffold inspect modules` commands
type ModulesOptions struct {
	// IncludeAll specifies if unnamed modules should be included in the output list
	IncludeAll bool
}

// ProfilesOptions holds flag values for various `skaffold inspect profiles` commands
type ProfilesOptions struct {
	// BuildEnv is the build-env filter for command output. One of: local, googleCloudBuild, cluster.
	BuildEnv BuildEnv
}

// BuildEnvOptions holds flag values for various `skaffold inspect build-env` commands
type BuildEnvOptions struct {
	// Profiles is the slice of profile names to activate.
	Profiles []string
	// Profile is a target profile to create or edit
	Profile string
	// Push specifies if images should be pushed to a registry.
	Push *bool
	// TryImportMissing specifies whether to attempt to import artifacts from Docker (either a local or remote registry) if not in the cache
	TryImportMissing *bool
	// UseDockerCLI specifies to use `docker` command-line interface instead of Docker Engine APIs
	UseDockerCLI *bool
	// UseBuildkit specifies to use Buildkit to build Docker images
	UseBuildkit *bool
	// ProjectID is the GCP project ID
	ProjectID string
	// DiskSizeGb is the disk size of the VM that runs the build
	DiskSizeGb int64
	// MachineType is the type of VM that runs the build
	MachineType string
	// Timeout is the build timeout (in seconds)
	Timeout string
	// Concurrency is the number of artifacts to build concurrently. 0 means "no-limit"
	Concurrency int
	// PullSecretPath is the path to the Google Cloud service account secret key file.
	PullSecretPath string
	// PullSecretName is the name of the Kubernetes secret for pulling base images
	// and pushing the final image.
	PullSecretName string
	// PullSecretMountPath is the path the pull secret will be mounted at within the running container.
	PullSecretMountPath string
	// Namespace is the Kubernetes namespace.
	Namespace string
	// DockerConfigPath is the path to the docker `config.json`.
	DockerConfigPath string
	// DockerConfigSecretName is the Kubernetes secret that contains the `config.json` Docker configuration.
	DockerConfigSecretName string
	// ServiceAccount describes the Kubernetes service account to use for the pod.
	ServiceAccount string
	// RunAsUser defines the UID to request for running the container.
	RunAsUser int64
	// RandomPullSecret adds a random UUID postfix to the default name of the pull secret to facilitate parallel builds, e.g. kaniko-secretdocker-cfgfd154022-c761-416f-8eb3-cf8258450b85.
	RandomPullSecret bool
	// RandomDockerConfigSecret adds a random UUID postfix to the default name of the docker secret to facilitate parallel builds, e.g. docker-cfgfd154022-c761-416f-8eb3-cf8258450b85.
	RandomDockerConfigSecret bool
	// Logging specifies the logging mode.
	Logging string
	// LogStreamingOption specifies the behavior when writing build logs to Google Cloud Storage.
	LogStreamingOption string
	// WorkerPool configures a pool of workers to run the build.
	WorkerPool string
}

type BuildEnv string

var (
	GetConfigSet = parser.GetConfigSet
	BuildEnvs    = struct {
		Unspecified      BuildEnv
		Local            BuildEnv
		GoogleCloudBuild BuildEnv
		Cluster          BuildEnv
	}{
		Local: "local", GoogleCloudBuild: "googleCloudBuild", Cluster: "cluster",
	}
)

func GetBuildEnv(t *latestV2.BuildType) BuildEnv {
	switch {
	case t.Cluster != nil:
		return BuildEnvs.Cluster
	case t.GoogleCloudBuild != nil:
		return BuildEnvs.GoogleCloudBuild
	default:
		return BuildEnvs.Local
	}
}
