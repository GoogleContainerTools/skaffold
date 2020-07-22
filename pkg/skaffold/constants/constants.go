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
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const (
	// DefaultLogLevel is the default global verbosity
	DefaultLogLevel = logrus.WarnLevel

	// DefaultDockerfilePath is the dockerfile path is given relative to the
	// context directory
	DefaultDockerfilePath = "Dockerfile"

	DefaultMinikubeContext         = "minikube"
	DefaultDockerForDesktopContext = "docker-for-desktop"
	DefaultDockerDesktopContext    = "docker-desktop"
	GCSBucketSuffix                = "_cloudbuild"

	HelmOverridesFilename = "skaffold-overrides.yaml"

	DefaultKustomizationPath = "."

	DefaultKanikoImage                  = "gcr.io/kaniko-project/executor:v0.20.0@sha256:f9a4a760166682c7c7aeda3cc263570682e00848ab47737ed8ffcc3abd2da6c3"
	DefaultKanikoSecretName             = "kaniko-secret"
	DefaultKanikoTimeout                = "20m"
	DefaultKanikoContainerName          = "kaniko"
	DefaultKanikoEmptyDirName           = "kaniko-emptydir"
	DefaultKanikoEmptyDirMountPath      = "/kaniko/buildcontext"
	DefaultKanikoCacheDirName           = "kaniko-cache"
	DefaultKanikoCacheDirMountPath      = "/cache"
	DefaultKanikoDockerConfigSecretName = "docker-cfg"
	DefaultKanikoDockerConfigPath       = "/kaniko/.docker"
	DefaultKanikoSecretMountPath        = "/secret"

	DefaultBusyboxImage = "busybox"

	// DefaultDebugHelpersRegistry is the default location used for the helper images for `debug`.
	DefaultDebugHelpersRegistry = "gcr.io/gcp-dev-tools/duct-tape"

	DefaultSkaffoldDir = ".skaffold"
	DefaultCacheFile   = "cache"

	DefaultRPCPort     = 50051
	DefaultRPCHTTPPort = 50052

	DefaultPortForwardNamespace = "default"
	DefaultPortForwardAddress   = "127.0.0.1"

	DefaultProjectDescriptor = "project.toml"

	LeeroyAppResponse = "leeroooooy app!!\n"
)

var (
	Pod     latest.ResourceType = "pod"
	Service latest.ResourceType = "service"

	DefaultLocalConcurrency = 1
)

var (
	// DeprecatedImages is an environment variable key, whose value is an array of fully qualified image names passed in to a custom build script.
	DeprecatedImages = "IMAGES"

	// Image is an environment variable key, whose value is the fully qualified image name passed in to a custom build script.
	Image = "IMAGE"

	// PushImage lets the custom build script know if the image is expected to be pushed to a remote registry
	PushImage = "PUSH_IMAGE"

	// BuildContext is the absolute path to a directory this artifact is meant to be built from for custom artifacts
	BuildContext = "BUILD_CONTEXT"

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
