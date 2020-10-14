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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShowAIError(t *testing.T) {
	tests := []struct {
		description string
		opts        config.SkaffoldOptions
		context     *config.ContextConfig
		err         error
		expected    string
	}{
		{
			description: "Push access denied when neither default repo or global config is defined",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{},
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Trying running with `--default-repo` flag.",
		},
		{
			description: "Push access denied when default repo is defined",
			opts:        config.SkaffoldOptions{DefaultRepo: stringOrUndefined("gcr.io/test")},
			context:     &config.ContextConfig{},
			err:         fmt.Errorf("skaffold build failed: could not push image image1 : denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your `--default-repo` value or try `gcloud auth configure-docker`.",
		},
		{
			description: "Push access denied when global repo is defined",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your default-repo setting in skaffold config or try `docker login`.",
		},
		{
			description: "unknown project error",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("build failed: could not push image: unknown: Project"),
			expected:    "Build Failed. Check your GCR project.",
		},
		{
			description: "unknown error",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("build failed: something went wrong"),
			expected:    "no suggestions found",
		},
		{
			description: "build error when docker is not running with minikube local cluster",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err: fmt.Errorf(`creating runner: creating builder: getting docker client: getting minikube env: running [/Users/tejaldesai/Downloads/google-cloud-sdk2/bin/minikube docker-env --shell none -p minikube]
 - stdout: "\n\n"
 - stderr: "! Executing \"docker container inspect minikube --format={{.State.Status}}\" took an unusually long time: 7.36540945s\n* Restarting the docker service may improve performance.\nX Exiting due to GUEST_STATUS: state: unknown state \"minikube\": docker container inspect minikube --format=: exit status 1\nstdout:\n\n\nstderr:\nCannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n\n* \n* If the above advice does not help, please let us know: \n  - https://github.com/kubernetes/minikube/issues/new/choose\n"
 - cause: exit status 80`),
			expected: "Build Failed. Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Please check if docker is running.",
		},
		{
			description: "build error when docker is not running and deploying to GKE",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf(`exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon at tcp://127.0.0.1:32770. Is the docker daemon running?`),
			expected:    "Build Failed. Cannot connect to the Docker daemon at tcp://127.0.0.1:32770. Please check if docker is running.",
		},

		{
			description: "build error when docker is not and no host information",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			// See https://github.com/moby/moby/blob/master/client/errors.go#L20
			err:      fmt.Errorf(`exiting dev mode because first build failed: docker build: Cannot connect to the Docker daemon. Is the docker daemon running on this host?`),
			expected: "Build Failed. Cannot connect to the Docker daemon. Please check if docker is running.",
		},
		{
			description: "build cancelled",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			// See https://github.com/moby/moby/blob/master/client/errors.go#L20
			err:      fmt.Errorf(`docker build: error during connect: Post \"https://127.0.0.1:32770/v1.24/build?buildargs=:  context canceled`),
			expected: "Build Cancelled.",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getConfigForCurrentContext, func(string) (*config.ContextConfig, error) {
				return test.context, nil
			})
			skaffoldOpts = test.opts
			actual := ShowAIError(test.err)
			t.CheckDeepEqual(test.expected, actual.Error())
		})
	}
}

func stringOrUndefined(s string) config.StringOrUndefined {
	c := &config.StringOrUndefined{}
	c.Set(s)
	return *c
}
