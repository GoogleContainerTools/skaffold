/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestMakeAuthSuggestionsForRepo(t *testing.T) {
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_DOCKER_AUTH_CONFIGURE,
		Action:         "try `docker login`",
	}, makeAuthSuggestionsForRepo(""), protocmp.Transform())
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
		Action:         "try `gcloud auth configure-docker`",
	}, makeAuthSuggestionsForRepo("gcr.io/test"), protocmp.Transform())
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
		Action:         "try `gcloud auth configure-docker`",
	}, makeAuthSuggestionsForRepo("eu.gcr.io/test"), protocmp.Transform())
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
		Action:         "try `gcloud auth configure-docker`",
	}, makeAuthSuggestionsForRepo("us-docker.pkg.dev/k8s-skaffold/skaffold"), protocmp.Transform())
}

func TestBuildProblems(t *testing.T) {
	tests := []struct {
		description string
		context     config.ContextConfig
		optRepo     string
		err         error
		expected    string
		expectedAE  *proto.ActionableErr
	}{
		{
			description: "Push access denied when neither default repo or global config is defined",
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Trying running with `--default-repo` flag.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED,
				Message: "skaffold build failed: could not push image: denied: push access to resource",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_ADD_DEFAULT_REPO,
					Action:         "Trying running with `--default-repo` flag",
				},
				}},
		},
		{
			description: "Push access denied when default repo is defined",
			optRepo:     "gcr.io/test",
			err:         fmt.Errorf("skaffold build failed: could not push image image1 : denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your `--default-repo` value or try `gcloud auth configure-docker`.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED,
				Message: "skaffold build failed: could not push image image1 : denied: push access to resource",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO,
					Action:         "Check your `--default-repo` value",
				}, {
					SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
					Action:         "try `gcloud auth configure-docker`",
				},
				},
			},
		},
		{
			description: "Push access denied when global repo is defined",
			context:     config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your default-repo setting in skaffold config or try `docker login`.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED,
				Message: "skaffold build failed: could not push image: denied: push access to resource",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO_GLOBAL_CONFIG,
					Action:         "Check your default-repo setting in skaffold config",
				}, {
					SuggestionCode: proto.SuggestionCode_DOCKER_AUTH_CONFIGURE,
					Action:         "try `docker login`",
				},
				},
			},
		},
		{
			description: "unknown project error",
			err:         fmt.Errorf("build failed: could not push image: unknown: Project test"),
			expected:    "Build Failed. could not push image: unknown: Project test. Check your GCR project.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_PROJECT_NOT_FOUND,
				Message: "build failed: could not push image: unknown: Project test",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_GCLOUD_PROJECT,
					Action:         "Check your GCR project",
				},
				},
			},
		},
		{
			description: "build error when docker is not running with minikube local cluster",
			err: fmt.Errorf(`creating runner: creating builder: getting docker client: getting minikube env: running [/Users/tejaldesai/Downloads/google-cloud-sdk2/bin/minikube docker-env --shell none -p minikube]
 - stdout: "\n\n"
 - stderr: "! Executing \"docker container inspect minikube --format={{.State.Status}}\" took an unusually long time: 7.36540945s\n* Restarting the docker service may improve performance.\nX Exiting due to GUEST_STATUS: state: unknown state \"minikube\": docker container inspect minikube --format=: exit status 1\nstdout:\n\n\nstderr:\nCannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n\n* \n* If the above advice does not help, please let us know: \n  - https://github.com/kubernetes/minikube/issues/new/choose\n"
 - cause: exit status 80`),
			expected: "Build Failed. Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Check if docker is running.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_DOCKER_DAEMON_NOT_RUNNING,
				Message: "creating runner: creating builder: getting docker client: getting minikube env: running [/Users/tejaldesai/Downloads/google-cloud-sdk2/bin/minikube docker-env --shell none -p minikube]\n - stdout: \"\\n\\n\"\n - stderr: \"! Executing \\\"docker container inspect minikube --format={{.State.Status}}\\\" took an unusually long time: 7.36540945s\\n* Restarting the docker service may improve performance.\\nX Exiting due to GUEST_STATUS: state: unknown state \\\"minikube\\\": docker container inspect minikube --format=: exit status 1\\nstdout:\\n\\n\\nstderr:\\nCannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\\n\\n* \\n* If the above advice does not help, please let us know: \\n  - https://github.com/kubernetes/minikube/issues/new/choose\\n\"\n - cause: exit status 80",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
					Action:         "Check if docker is running",
				},
				},
			},
		},
		{
			description: "build error when docker is not running and deploying to GKE",
			err:         fmt.Errorf(`exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon at tcp://127.0.0.1:32770. Is the docker daemon running?`),
			expected:    "Build Failed. Cannot connect to the Docker daemon at tcp://127.0.0.1:32770. Check if docker is running.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_DOCKER_DAEMON_NOT_RUNNING,
				Message: "exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon at tcp://127.0.0.1:32770. Is the docker daemon running?",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
					Action:         "Check if docker is running",
				},
				},
			},
		},

		{
			description: "build error when docker is not and no host information",
			// See https://github.com/moby/moby/blob/master/client/errors.go#L20
			err:      fmt.Errorf(`exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon. Is the docker daemon running on this host?`),
			expected: "Build Failed. Cannot connect to the Docker daemon. Check if docker is running.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_DOCKER_DAEMON_NOT_RUNNING,
				Message: "exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon. Is the docker daemon running on this host?",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
					Action:         "Check if docker is running",
				},
				},
			},
		},
		{
			description: "build cancelled",
			// See https://github.com/moby/moby/blob/master/client/errors.go#L20
			err:      fmt.Errorf(`docker build: error during connect: Post \"https://127.0.0.1:32770/v1.24/build?buildargs=:  context canceled`),
			expected: "Build Cancelled",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_CANCELLED,
				Message: `docker build: error during connect: Post \"https://127.0.0.1:32770/v1.24/build?buildargs=:  context canceled`,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getConfigForCurrentContext, func(string) (*config.ContextConfig, error) {
				return &test.context, nil
			})
			t.Override(&sErrors.GetProblemCatalogCopy, func() sErrors.ProblemCatalog {
				pc := sErrors.NewProblemCatalog()
				pc.AddPhaseProblems(constants.Build, problems)
				return pc
			})
			cfg := mockConfig{optRepo: test.optRepo}
			actual := sErrors.ShowAIError(&cfg, test.err)
			t.CheckDeepEqual(test.expected, actual.Error())
			actualAE := sErrors.ActionableErr(&cfg, constants.Build, test.err)
			t.CheckDeepEqual(test.expectedAE, actualAE, protocmp.Transform())
		})
	}
}
