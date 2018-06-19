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

	// DefaultKanikoImage is v0.1.0
	DefaultKanikoImage = "gcr.io/kaniko-project/executor:v0.1.0@sha256:501056bf52f3a96f151ccbeb028715330d5d5aa6647e7572ce6c6c55f91ab374"
)

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
