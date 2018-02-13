/*
Copyright 2018 Google LLC

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
	// For alpha releases, the default log level should be 'info'
	DefaultLogLevel = logrus.InfoLevel

	// The dockerfile path is given relative to the context directory
	DefaultDockerfilePath = "Dockerfile"

	DefaultTagStrategy = TagStrategyGitCommit

	// TagStrategySha256 uses the checksum of the built artifact as the tag
	TagStrategySha256    = "sha256"
	TagStrategyGitCommit = "gitCommit"

	DefaultMinikubeContext = "minikube"
)
