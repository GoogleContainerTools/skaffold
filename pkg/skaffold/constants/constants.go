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
	"fmt"
	"runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/sirupsen/logrus"
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

	DefaultKanikoImage                  = "gcr.io/kaniko-project/executor:v0.9.0@sha256:d9fe474f80b73808dc12b54f45f5fc90f7856d9fc699d4a5e79d968a1aef1a72"
	DefaultKanikoSecretName             = "kaniko-secret"
	DefaultKanikoTimeout                = "20m"
	DefaultKanikoContainerName          = "kaniko"
	DefaultKanikoEmptyDirName           = "kaniko-emptydir"
	DefaultKanikoEmptyDirMountPath      = "/kaniko/buildcontext"
	DefaultKanikoDockerConfigSecretName = "docker-cfg"
	DefaultKanikoDockerConfigPath       = "/kaniko/.docker"

	DefaultBusyboxImage = "busybox"

	UpdateCheckEnvironmentVariable = "SKAFFOLD_UPDATE_CHECK"

	DefaultCloudBuildDockerImage = "gcr.io/cloud-builders/docker"
	DefaultCloudBuildMavenImage  = "gcr.io/cloud-builders/mvn@sha256:0ec283f2ee1ab1d2ac779dcbb24bddaa46275aec7088cc10f2926b4ea0fcac9b"
	DefaultCloudBuildGradleImage = "gcr.io/cloud-builders/gradle"

	DefaultSkaffoldDir = ".skaffold"
	DefaultCacheFile   = "cache"

	SkaffoldPluginKey       = "SKAFFOLD_PLUGIN_KEY"
	SkaffoldPluginValue     = "1337"
	SkaffoldPluginName      = "SKAFFOLD_PLUGIN_NAME"
	DockerBuilderPluginName = "docker"

	DefaultRPCPort     = 50051
	DefaultRPCHTTPPort = 50052
)

var (
	GoogleCloudBuild latest.ExecEnvironment = "googleCloudBuild"
	Local            latest.ExecEnvironment = "local"
)

var DefaultKubectlManifests = []string{"k8s/*.yaml"}

var LatestDownloadURL = fmt.Sprintf("https://storage.googleapis.com/skaffold/releases/latest/skaffold-%s-%s", runtime.GOOS, runtime.GOARCH)

var Labels = struct {
	TagPolicy        string
	Deployer         string
	Builder          string
	DockerAPIVersion string
	DefaultLabels    map[string]string
}{
	DefaultLabels: map[string]string{
		"deployed-with": "skaffold",
	},
	TagPolicy:        "skaffold-tag-policy",
	Deployer:         "skaffold-deployer",
	Builder:          "skaffold-builder",
	DockerAPIVersion: "docker-api-version",
}
