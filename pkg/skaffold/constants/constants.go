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

package constants

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

const (
	// These are phases in Skaffold
	DevLoop     = Phase("DevLoop")
	Init        = Phase("Init")
	Build       = Phase("Build")
	Test        = Phase("Test")
	Render      = Phase("Render")
	Deploy      = Phase("Deploy")
	Verify      = Phase("Verify")
	StatusCheck = Phase("StatusCheck")
	PortForward = Phase("PortForward")
	Sync        = Phase("Sync")
	DevInit     = Phase("DevInit")
	Cleanup     = Phase("Cleanup")

	// DefaultDockerfilePath is the dockerfile path is given relative to the
	// context directory
	DefaultDockerfilePath = "Dockerfile"

	DefaultMinikubeContext         = "minikube"
	DefaultDockerForDesktopContext = "docker-for-desktop"
	DefaultDockerDesktopContext    = "docker-desktop"
	GCSBucketSuffix                = "_cloudbuild"

	HelmOverridesFilename = "skaffold-overrides.yaml"

	DefaultKustomizationPath = "."

	DefaultBusyboxImage = "gcr.io/k8s-skaffold/skaffold-helpers/busybox"

	// DefaultDebugHelpersRegistry is the default location used for the helper images for `debug`.
	DefaultDebugHelpersRegistry = "gcr.io/k8s-skaffold/skaffold-debug-support"

	DefaultSkaffoldDir = ".skaffold"
	DefaultCacheFile   = "cache"
	DefaultMetricFile  = "metrics"

	// SkaffoldEnvFile is the file that is parsed to set environment variables in the process
	SkaffoldEnvFile = "skaffold.env"

	DefaultPortForwardAddress = "127.0.0.1"

	DefaultProjectDescriptor = "project.toml"

	DefaultBuildpacksBuilderImage = "gcr.io/buildpacks/builder:v1"

	LeeroyAppResponse = "leeroooooy app!!\n"

	GithubIssueLink = "https://github.com/GoogleContainerTools/skaffold/issues/new"

	Windows = "windows"

	DefaultHydrationDir = ".kpt-pipeline"
	// HaTS is the HaTS Survey ID
	HaTS = "hats"

	// SubtaskIDNone is the value used for Event API messages when there is no
	// corresponding subtask
	SubtaskIDNone = "-1"
)

type Phase string

var (
	Pod     latest.ResourceType = "pod"
	Service latest.ResourceType = "service"

	DefaultLocalConcurrency = 1
)

var (
	// Image is an environment variable key, whose value is the fully qualified image name passed in to a custom build script.
	Image = "IMAGE"

	// PushImage lets the custom build script know if the image is expected to be pushed to a remote registry
	PushImage = "PUSH_IMAGE"

	// BuildContext is the absolute path to a directory this artifact is meant to be built from for custom artifacts
	BuildContext = "BUILD_CONTEXT"

	// SkipTest is Whether to skip the tests after building passing into a custom build script
	SkipTest = "SKIP_TEST"

	// Platforms is the set of platforms to build the image for.
	Platforms = "PLATFORMS"

	// KubeContext is the expected kubecontext to build an artifact with a custom build script on cluster
	KubeContext = "KUBE_CONTEXT"

	// Namespace is the expected namespace to build an artifact with a custom build script on cluster.
	Namespace = "NAMESPACE"

	// PullSecretName is the secret with authentication required to pull a base image/push the final image built on cluster.
	PullSecretName = "PULL_SECRET_NAME"

	// DockerConfigSecretName is the secret containing any required docker authentication for custom builds on cluster.
	DockerConfigSecretName = "DOCKER_CONFIG_SECRET_NAME"

	// Timeout is the amount of time an on cluster build is allowed to run.
	Timeout = "TIMEOUT"

	AllowedUsers = map[string]struct{}{
		"vsc":          {},
		"intellij":     {},
		"gcloud":       {},
		"cloud-deploy": {},
	}

	AllowedUserPattern = `^%v(\/.+)?$`

	KustomizeFilePaths = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}

	DefaultKanikoDigestFile = "/dev/termination-log"
)

var ImageRef = struct {
	Repo   string
	Tag    string
	Digest string
}{
	Repo:   "IMAGE_REPO",
	Tag:    "IMAGE_TAG",
	Digest: "IMAGE_DIGEST",
}
var DefaultKubectlManifests = []string{"k8s/*.yaml"}

var Labels = struct {
	TagPolicy        string
	Deployer         string
	Builder          string
	DockerAPIVersion string
}{
	TagPolicy:        "skaffold.dev/tag-policy",
	Deployer:         "skaffold.dev/deployer",
	Builder:          "skaffold.dev/builder",
	DockerAPIVersion: "skaffold.dev/docker-api-version",
}

const (
	// RemoteDigestSource skips builds and resolves the digest of images by tag from the remote registry.
	RemoteDigestSource = "remote"
	// TagDigestSource to  uses tags directly from the build.
	TagDigestSource = "tag"
	// NoneDigestSourceSet uses tags directly from the Kubernetes manifests.
	NoneDigestSource = "none"
)
