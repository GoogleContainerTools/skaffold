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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewEnvClient(t *testing.T) {
	tests := []struct {
		description string
		envs        map[string]string
		shouldErr   bool
	}{
		{
			description: "get client",
			envs: map[string]string{
				"DOCKER_HOST": "http://127.0.0.1:8080",
			},
		},
		{
			description: "invalid cert path",
			envs: map[string]string{
				"DOCKER_HOST":      "http://127.0.0.1:8080",
				"DOCKER_CERT_PATH": "invalid/cert/path",
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(test.envs)

			env, _, err := newEnvAPIClient()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, []string(nil), env)
		})
	}
}

func TestNewMinikubeImageAPIClient(t *testing.T) {
	tests := []struct {
		description string
		env         string
		expectedEnv []string
		shouldErr   bool
	}{
		{
			description: "correct client",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`,
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "bad certificate",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=bad/cert/path
DOCKER_API_VERSION=1.23`,
			shouldErr: true,
		},
		{
			description: "missing host env, no error",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`,
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "missing version env, no error",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=testdata`,
			expectedEnv: []string{"DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "bad url",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_HOST=badurl
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`,
			shouldErr: true,
		},
		{
			description: "bad env output, should fallback to host docker",
			env: `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080=toomanyvalues
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				"minikube docker-env --shell none",
				test.env,
			))

			env, _, err := newMinikubeAPIClient()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedEnv, env)
		})
	}
}
