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

package constants

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultLogLevel is the default global verbosity
	DefaultLogLevel = logrus.WarnLevel

	// DefaultDockerfilePath is the dockerfile path is given relative to the
	// context directory
	DefaultDockerfilePath = "Dockerfile"

	DefaultDevTagStrategy = TagStrategySha256
	DefaultRunTagStrategy = TagStrategyGitCommit

	// TagStrategySha256 uses the checksum of the built artifact as the tag
	TagStrategySha256    = "sha256"
	TagStrategyGitCommit = "gitCommit"

	DefaultMinikubeContext         = "minikube"
	DefaultDockerForDesktopContext = "docker-for-desktop"
	GCSBucketSuffix                = "_cloudbuild"

	HelmOverridesFilename = "skaffold-overrides.yaml"

	DefaultKustomizationPath = "."

	DefaultKanikoImage      = "gcr.io/kaniko-project/executor:v0.4.0@sha256:0bbaa4859eec9796d32ab45e6c1627562dbc7796e40450295b9604cd3f4197af"
	DefaultKanikoSecretName = "kaniko-secret"
	DefaultKanikoTimeout    = "20m"

	UpdateCheckEnvironmentVariable = "SKAFFOLD_UPDATE_CHECK"

	DefaultCloudBuildDockerImage = "gcr.io/cloud-builders/docker"
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
