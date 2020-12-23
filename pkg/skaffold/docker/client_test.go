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
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
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
		command     util.Command
		expectedEnv []string
		shouldErr   bool
	}{
		{
			description: "correct client",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`),
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "correct client - work around minikube #8615",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23

# To point your shell to minikube's docker-daemon, run:
# eval $(minikube -p minikube docker-env)
`),
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "bad certificate",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=bad/cert/path
DOCKER_API_VERSION=1.23`),
			shouldErr: true,
		},
		{
			description: "missing host env, no error",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`),
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "missing version env, no error",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080
DOCKER_CERT_PATH=testdata`),
			expectedEnv: []string{"DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "bad url",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=badurl
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`),
			shouldErr: true,
		},
		{
			description: "allow `=` in urls",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST=http://127.0.0.1:8080?k=v
DOCKER_CERT_PATH=testdata
DOCKER_API_VERSION=1.23`),
			expectedEnv: []string{"DOCKER_API_VERSION=1.23", "DOCKER_CERT_PATH=testdata", "DOCKER_HOST=http://127.0.0.1:8080?k=v", "DOCKER_TLS_VERIFY=1"},
		},
		{
			description: "bad env output",
			command: testutil.CmdRunOut("minikube docker-env --shell none -p minikube", `DOCKER_TLS_VERIFY=1
DOCKER_HOST`),
			shouldErr: true,
		},
		{
			description: "command error",
			command:     testutil.CmdRunOutErr("minikube docker-env --shell none -p minikube", "", errors.New("fail")),
			shouldErr:   true,
		},
		{
			description: "minikube exit code 64 (minikube < 1.13.0) - fallback to host docker",
			command:     testutil.CmdRunOutErr("minikube docker-env --shell none -p minikube", "", fmt.Errorf("fail: %w", &oldBadUsageErr{})),
		},
		{
			description: "minikube exit code 51 (minikube >= 1.13.0) - fallback to host docker",
			command:     testutil.CmdRunOutErr("minikube docker-env --shell none -p minikube", "", fmt.Errorf("fail: %w", &driverConflictErr{})),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			t.Override(&cluster.GetClient, func() cluster.Client { return fakeMinikubeClient{} })

			env, _, err := newMinikubeAPIClient("minikube")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedEnv, env)
		})
	}
}

// minikube < 1.13.0 returns exit code 64 (BadUsage) on `minikube docker-env` with driver `none`
type oldBadUsageErr struct{}

func (e *oldBadUsageErr) Error() string { return "bad usage" }
func (e *oldBadUsageErr) ExitCode() int { return 64 }

// minikube >= 1.13.0 returns exit code 51 (ExDriverConfict) on `minikube docker-env` with driver `none`
type driverConflictErr struct{}

func (e *driverConflictErr) Error() string { return "driver conflict" }
func (e *driverConflictErr) ExitCode() int { return 51 }

type fakeMinikubeClient struct{}

func (fakeMinikubeClient) IsMinikube(string) bool { return false }
func (fakeMinikubeClient) MinikubeExec(arg ...string) (*exec.Cmd, error) {
	return exec.Command("minikube", arg...), nil
}
