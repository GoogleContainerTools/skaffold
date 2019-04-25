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

package local

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRetrieveEnv(t *testing.T) {
	tests := []struct {
		description string
		tag         string
		pushImage   bool
		environ     []string
		dockerEnv   []string
		expected    []string
	}{

		{
			description: "make sure tags are correct",
			tag:         "gcr.io/image/tag:mytag",
			environ:     nil,
			expected:    []string{"IMAGE_NAME=gcr.io/image/tag:mytag", "PUSH_IMAGE=false"},
		}, {
			description: "make sure environ is correctly applied",
			tag:         "gcr.io/image/tag:anothertag",
			environ:     []string{"PATH=/path", "HOME=/root"},
			expected:    []string{"IMAGE_NAME=gcr.io/image/tag:anothertag", "PUSH_IMAGE=false", "PATH=/path", "HOME=/root"},
		}, {
			description: "make sure docker env is correctly applied",
			tag:         "gcr.io/image/docker:tag",
			dockerEnv:   []string{"DOCKER_API_VERSION=1.3", "DOCKER_CERT_PATH=/home/.minikube/certs"},
			expected:    []string{"IMAGE_NAME=gcr.io/image/docker:tag", "PUSH_IMAGE=false", "DOCKER_API_VERSION=1.3", "DOCKER_CERT_PATH=/home/.minikube/certs"},
		}, {
			description: "push image is true",
			tag:         "gcr.io/image/push:tag",
			pushImage:   true,
			expected:    []string{"IMAGE_NAME=gcr.io/image/push:tag", "PUSH_IMAGE=true"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			initial := environ
			defer func() {
				environ = initial
			}()
			environ = func() []string {
				return test.environ
			}
			actual := retrieveEnv(test.tag, test.pushImage, test.dockerEnv)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
