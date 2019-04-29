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

package custom

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRetrieveEnv(t *testing.T) {
	tests := []struct {
		description  string
		tag          string
		pushImages   bool
		buildContext string
		api          *testutil.FakeAPIClient
		artifact     *latest.Artifact
		dockerEnv    []string
		environ      []string
		expected     []string
	}{

		{
			description:  "make sure tags are correct",
			tag:          "gcr.io/image/tag:mytag",
			environ:      nil,
			buildContext: "/some/path",
			expected:     []string{"BUILD_CONTEXT=/some/path", "IMAGES=gcr.io/image/tag:mytag", "PUSH_IMAGE=false"},
		}, {
			description:  "make sure environ is correctly applied",
			tag:          "gcr.io/image/tag:anothertag",
			environ:      []string{"PATH=/path", "HOME=/root"},
			buildContext: "/some/path",
			expected:     []string{"BUILD_CONTEXT=/some/path", "HOME=/root", "IMAGES=gcr.io/image/tag:anothertag", "PATH=/path", "PUSH_IMAGE=false"},
		},
		{
			description: "make sure docker env is correctly applied",
			tag:         "gcr.io/image/docker:tag",
			dockerEnv:   []string{"DOCKER_API_VERSION=1.3", "DOCKER_CERT_PATH=/home/.minikube/certs"},
			expected:    []string{"BUILD_CONTEXT=", "DOCKER_API_VERSION=1.3", "DOCKER_CERT_PATH=/home/.minikube/certs", "IMAGES=gcr.io/image/docker:tag", "PUSH_IMAGE=false"},
		}, {
			description: "push image is true",
			tag:         "gcr.io/image/push:tag",
			pushImages:  true,
			expected:    []string{"BUILD_CONTEXT=", "IMAGES=gcr.io/image/push:tag", "PUSH_IMAGE=true"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			initialEnviron := environ
			defer func() {
				environ = initialEnviron
			}()
			environ = func() []string {
				return test.environ
			}

			initialBuildContext := buildContext
			defer func() {
				buildContext = initialBuildContext
			}()
			buildContext = func(_ string) (string, error) {
				return test.buildContext, nil
			}

			client := docker.NewLocalDaemon(test.api, test.dockerEnv, false, nil)
			artifactBuilder := NewArtifactBuilder(test.pushImages, "", nil, client)
			actual, err := artifactBuilder.retrieveEnv(&latest.Artifact{}, test.tag)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
		})
	}
}
