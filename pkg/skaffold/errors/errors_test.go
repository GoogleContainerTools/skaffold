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

package errors

import (
	"fmt"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShowAIError(t *testing.T) {
	tests := []struct {
		description string
		opts        config.SkaffoldOptions
		phase       Phase
		context     *config.ContextConfig
		err         error
		expected    string
		expectedAE  *proto.ActionableErr
	}{
		{
			description: "Push access denied when neither default repo or global config is defined",
			context:     &config.ContextConfig{},
			phase:       Build,
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
			opts:        config.SkaffoldOptions{DefaultRepo: stringOrUndefined("gcr.io/test")},
			phase:       Build,
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
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			phase:       Build,
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
			phase:       Build,
			err:         fmt.Errorf("build failed: could not push image: unknown: Project"),
			expected:    "Build Failed. Check your GCR project.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_PROJECT_NOT_FOUND,
				Message: "build failed: could not push image: unknown: Project",
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
			phase:    Build,
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
			phase:       Build,
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
			phase:    Build,
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
			phase:    Build,
			expected: "Build Cancelled.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_BUILD_CANCELLED,
				Message: `docker build: error during connect: Post \"https://127.0.0.1:32770/v1.24/build?buildargs=:  context canceled`,
			},
		},
		// unknown errors case
		{
			description: "build unknown error",
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			phase:       Build,
			err:         fmt.Errorf("build failed: something went wrong"),
			expected:    "build failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_BUILD_UNKNOWN,
				Message:     "build failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "deploy unknown error",
			phase:       Deploy,
			err:         fmt.Errorf("deploy failed: something went wrong"),
			expected:    "deploy failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_DEPLOY_UNKNOWN,
				Message:     "deploy failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "file sync unknown error",
			phase:       FileSync,
			err:         fmt.Errorf("sync failed: something went wrong"),
			expected:    "sync failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_SYNC_UNKNOWN,
				Message:     "sync failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "init unknown error",
			phase:       Init,
			err:         fmt.Errorf("init failed: something went wrong"),
			expected:    "init failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_INIT_UNKNOWN,
				Message:     "init failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "cleanup unknown error",
			phase:       Cleanup,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_CLEANUP_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "status check unknown error",
			phase:       StatusCheck,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_STATUSCHECK_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "dev init unknown error",
			phase:       DevInit,
			err:         fmt.Errorf("failed: something went wrong"),
			expected:    "failed: something went wrong",
			expectedAE: &proto.ActionableErr{
				ErrCode:     proto.StatusCode_DEVINIT_UNKNOWN,
				Message:     "failed: something went wrong",
				Suggestions: reportIssueSuggestion(config.SkaffoldOptions{}),
			},
		},
		{
			description: "deploy failed",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{},
			phase:       Deploy,
			err:         fmt.Errorf(`exiting dev mode because first deploy failed: unable to connect to Kubernetes: Get "https://192.168.64.3:8443/version?timeout=32s": net/http: TLS handshake timeout`),
			expected:    "Deploy Failed. Could not connect to cluster test_cluster due to \"https://192.168.64.3:8443/version?timeout=32s\": net/http: TLS handshake timeout. Check your connection for the cluster.",
			expectedAE: &proto.ActionableErr{
				ErrCode: proto.StatusCode_DEPLOY_CLUSTER_CONNECTION_ERR,
				Message: "exiting dev mode because first deploy failed: unable to connect to Kubernetes: Get \"https://192.168.64.3:8443/version?timeout=32s\": net/http: TLS handshake timeout",
				Suggestions: []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
					Action:         "Check your connection for the cluster",
				}},
			},
		},
	}
	for _, test := range append(tests, initTestCases...) {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getConfigForCurrentContext, func(string) (*config.ContextConfig, error) {
				return test.context, nil
			})
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "test_cluster"})
			skaffoldOpts = test.opts
			actual := ShowAIError(test.err)
			t.CheckDeepEqual(test.expected, actual.Error())
			actualAE := ActionableErr(test.phase, test.err)
			t.CheckDeepEqual(test.expectedAE, actualAE)
		})
	}
}

func TestIsOldImageManifestProblem(t *testing.T) {
	tests := []struct {
		description string
		command     string
		err         error
		expectedMsg string
		expected    bool
	}{
		{
			description: "dev command older manifest with image name",
			command:     "dev",
			err:         fmt.Errorf(`listing files: parsing ONBUILD instructions: retrieving image "library/ruby:2.3.0": unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image library/ruby:2.3.0 pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "dev command older manifest without image name",
			command:     "dev",
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "dev command with random name",
			command:     "dev",
			err:         fmt.Errorf(`listing files: parsing ONBUILD instructions: retrieve image "noimage" image does not exits`),
		},
		{
			description: "debug command older manifest",
			command:     "debug",
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expectedMsg: "Could not retrieve image pushed with the deprecated manifest v1. Ignoring files dependencies for all ONBUILD triggers. To avoid, hit Cntrl-C and run `docker pull` to fetch the specified image and retry.",
			expected:    true,
		},
		{
			description: "build command older manifest",
			command:     "build",
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
		{
			description: "run command older manifest",
			command:     "run",
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
		{
			description: "deploy command older manifest",
			command:     "deploy",
			err:         fmt.Errorf(`unsupported MediaType: "application/vnd.docker.distribution.manifest.v1+prettyjws", see https://github.com/google/go-containerregistry/issues/377`),
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			skaffoldOpts = config.SkaffoldOptions{
				Command: test.command,
			}
			actualMsg, actual := IsOldImageManifestProblem(test.err)
			fmt.Println(actualMsg)
			t.CheckDeepEqual(test.expectedMsg, actualMsg)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func stringOrUndefined(s string) config.StringOrUndefined {
	c := &config.StringOrUndefined{}
	c.Set(s)
	return *c
}
