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

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestLocalVerifyPassingTestsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort,
		"--event-log-file", logFile, "--env-file", "verify.env").InDir("testdata/verify-succeed").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, false, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)
	testutil.CheckContains(t, "alpine-1", logs)
	testutil.CheckContains(t, "alpine-2", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that SUCCEEDED event is found where expected
}

func TestVerifyWithNotCreatedNetwork(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	// `--default-repo=` is used to cancel the default repo that is set by default.
	logs, err := skaffold.Verify("--default-repo=", "--env-file", "verify.env", "--docker-network", "not-created-network").InDir("testdata/verify-succeed").RunWithCombinedOutput(t)
	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "network not-created-network not found", string(logs))
}

func TestLocalVerifyOneTestFailsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort,
		"--event-log-file", logFile, "--env-file", "verify.env").InDir("testdata/verify-fail").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that FAILED event is found where expected
}

func TestVerifyNoTestsFails(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=").InDir("testdata/verify-no-tests").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "verify command expects non-zero number of test cases", logs)
}

func TestKubernetesJobVerifyPassingTestsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort,
		"--event-log-file", logFile, "--env-file", "verify.env").InDir("testdata/verify-succeed-k8s").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, false, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)
	testutil.CheckContains(t, "alpine-1", logs)
	testutil.CheckContains(t, "alpine-2", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that SUCCEEDED event is found where expected
}

func TestKubernetesJobVerifyOneTestFailsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort,
		"--event-log-file", logFile, "--env-file", "verify.env").InDir("testdata/verify-fail-k8s").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that FAILED event is found where expected
}

func TestNoDuplicateLogsLocal(t *testing.T) {
	tests := []struct {
		description        string
		dir                string
		profile            string
		shouldErr          bool
		expectedUniqueLogs []string
	}{
		{
			description: "no duplicated logs in docker actions, success execution",
			dir:         "testdata/verify-succeed",
			profile:     "no-duplicated-logs",
			expectedUniqueLogs: []string{
				"[alpine-1] alpine-1",
				"[alpine-1] bye alpine-1",
			},
		},
		{
			description: "no duplicated logs in docker actions, fail execution",
			dir:         "testdata/verify-fail",
			profile:     "no-duplicated-logs",
			shouldErr:   true,
			expectedUniqueLogs: []string{
				"[alpine-1] alpine-1",
				"[alpine-1] bye alpine-1",
				"[alpine-2] alpine-2",
				"[alpine-2] bye alpine-2",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)

			args := []string{"-p", test.profile}
			out, err := skaffold.Verify(args...).InDir(test.dir).RunWithCombinedOutput(t.T)

			t.CheckError(test.shouldErr, err)

			logs := string(out)
			checkUniqueLogs(t, logs, test.expectedUniqueLogs)
		})
	}
}

func TestNoDuplicateLogsK8SJobs(t *testing.T) {
	tests := []struct {
		description        string
		dir                string
		profile            string
		shouldErr          bool
		expectedUniqueLogs []string
	}{
		{
			description: "no duplicated logs in k8s actions, success execution",
			dir:         "testdata/verify-succeed-k8s",
			profile:     "no-duplicated-logs",
			expectedUniqueLogs: []string{
				"[alpine-1] alpine-1",
				"[alpine-1] bye alpine-1",
			},
		},
		{
			description: "no duplicated logs in k8s actions, fail execution",
			dir:         "testdata/verify-fail-k8s",
			profile:     "no-duplicated-logs",
			shouldErr:   true,
			expectedUniqueLogs: []string{
				"[alpine-1] alpine-1",
				"[alpine-1] bye alpine-1",
				"[alpine-2] alpine-2",
				"[alpine-2] bye alpine-2",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)

			args := []string{"-p", test.profile}
			out, err := skaffold.Verify(args...).InDir(test.dir).RunWithCombinedOutput(t.T)

			t.CheckError(test.shouldErr, err)

			logs := string(out)
			checkUniqueLogs(t, logs, test.expectedUniqueLogs)
		})
	}
}

func TestTimeoutK8s(t *testing.T) {
	tests := []struct {
		description     string
		dir             string
		profile         string
		shouldErr       bool
		expectedLogs    []string
		notExpectedLogs []string
	}{
		{
			description: "K8s - One test fail due to timeout",
			dir:         "testdata/verify-fail-k8s",
			profile:     "fail-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`1 error(s) occurred:`,
				`* "alpine-3" running k8s job timed out after : 5s`,
			},
			notExpectedLogs: []string{
				`[alpine-3] bye alpine-3`,
			},
		},
		{
			description: "K8s - Two tests with different timeouts",
			dir:         "testdata/verify-fail-k8s",
			profile:     "fail-two-test-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`[alpine-4] alpine-4`,
				`[alpine-5] alpine-5`,
				`* "alpine-4" running k8s job timed out after : 6s`,
				`* "alpine-5" running k8s job timed out after : 5s`,
			},
			notExpectedLogs: []string{
				`[alpine-4] bye alpine-4`,
				`[alpine-5] bye alpine-5`,
			},
		},
		{
			description: "K8s - Two tests, one fail other succeed",
			dir:         "testdata/verify-fail-k8s",
			profile:     "fail-only-one-test-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`[alpine-6] alpine-6`,
				`[alpine-7] alpine-7`,
				`[alpine-7] bye alpine-7`,
				`* "alpine-6" running k8s job timed out after : 6s`,
			},
			notExpectedLogs: []string{
				`[alpine-6] bye alpine-6`,
			},
		},
		{
			description: "K8s - Two tests with timeouts, all succeed",
			dir:         "testdata/verify-succeed-k8s",
			profile:     "succeed-with-timeout",
			expectedLogs: []string{
				`[alpine-8] alpine-8`,
				`[alpine-8] bye alpine-8`,
				`[alpine-9] alpine-9`,
				`[alpine-9] bye alpine-9`,
			},
			notExpectedLogs: []string{
				`* "alpine-8" running k8s job timed out after : 20s`,
				`* "alpine-9" running k8s job timed out after : 25s`,
			},
		},
		{
			description: "K8s - Two tests, one with timeout, all succeed",
			dir:         "testdata/verify-succeed-k8s",
			profile:     "succeed-all-one-with-timeout",
			expectedLogs: []string{
				`[alpine-10] alpine-10`,
				`[alpine-10] bye alpine-10`,
				`[alpine-11] alpine-11`,
				`[alpine-11] bye alpine-11`,
			},
			notExpectedLogs: []string{
				`* "alpine-11" running k8s job timed out after : 25s`,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)

			args := []string{"-p", test.profile}
			out, err := skaffold.Verify(args...).InDir(test.dir).RunWithCombinedOutput(t.T)
			logs := string(out)

			t.CheckError(test.shouldErr, err)

			for _, el := range test.expectedLogs {
				t.CheckContains(el, logs)
			}

			for _, nel := range test.notExpectedLogs {
				testutil.CheckNotContains(t.T, nel, logs)
			}
		})
	}
}

func TestTimeoutDocker(t *testing.T) {
	tests := []struct {
		description     string
		dir             string
		profile         string
		shouldErr       bool
		expectedLogs    []string
		notExpectedLogs []string
	}{
		{
			description: "Docker - One test fail due to timeout",
			dir:         "testdata/verify-fail",
			profile:     "fail-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`1 error(s) occurred:`,
				`* verify test failed: "alpine-1" running container image "alpine:3.15.4" timed out after : 5s`,
			},
			notExpectedLogs: []string{
				`[alpine-1] bye alpine-1`,
			},
		},
		{
			description: "Docker - Two tests with different timeouts",
			dir:         "testdata/verify-fail",
			profile:     "fail-two-test-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`[alpine-2] alpine-2`,
				`[alpine-1] alpine-1`,
				`* verify test failed: "alpine-1" running container image "alpine:3.15.4" timed out after : 6s`,
				`* verify test failed: "alpine-2" running container image "alpine:3.15.4" timed out after : 5s`,
			},
			notExpectedLogs: []string{
				`[alpine-1] bye alpine-1`,
				`[alpine-2] bye alpine-2`,
			},
		},
		{
			description: "Docker - Two tests, one fail other succeed",
			dir:         "testdata/verify-fail",
			profile:     "fail-only-one-test-timeout",
			shouldErr:   true,
			expectedLogs: []string{
				`[alpine-1] alpine-1`,
				`[alpine-2] alpine-2`,
				`[alpine-2] bye alpine-2`,
				`* verify test failed: "alpine-1" running container image "alpine:3.15.4" timed out after : 6s`,
			},
			notExpectedLogs: []string{
				`[alpine-1] bye alpine-1`,
			},
		},
		{
			description: "Docker - Two tests with timeouts, all succeed",
			dir:         "testdata/verify-succeed",
			profile:     "succeed-with-timeout",
			expectedLogs: []string{
				`[alpine-1] alpine-1`,
				`[alpine-1] bye alpine-1`,
				`[alpine-2] alpine-2`,
				`[alpine-2] bye alpine-2`,
			},
			notExpectedLogs: []string{
				`* verify test failed: "alpine-1" running container image "alpine:3.15.4" timed out after : 20s`,
				`* verify test failed: "alpine-2" running container image "alpine:3.15.4" timed out after : 25s`,
			},
		},
		{
			description: "Docker - Two tests, one with timeout, all succeed",
			dir:         "testdata/verify-succeed",
			profile:     "succeed-all-one-with-timeout",
			expectedLogs: []string{
				`[alpine-1] alpine-1`,
				`[alpine-1] bye alpine-1`,
				`[alpine-2] alpine-2`,
				`[alpine-2] bye alpine-2`,
			},
			notExpectedLogs: []string{
				`* verify test failed: "alpine-2" running container image "alpine:3.15.4" timed out after : 25s`,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)

			args := []string{"-p", test.profile}
			out, err := skaffold.Verify(args...).InDir(test.dir).RunWithCombinedOutput(t.T)
			logs := string(out)

			t.CheckError(test.shouldErr, err)

			for _, el := range test.expectedLogs {
				t.CheckContains(el, logs)
			}

			for _, nel := range test.notExpectedLogs {
				testutil.CheckNotContains(t.T, nel, logs)
			}
		})
	}
}

func checkUniqueLogs(t *testutil.T, logs string, expectedUniqueLogs []string) {
	for _, uniqueLog := range expectedUniqueLogs {
		timesFound := strings.Count(logs, uniqueLog)
		if timesFound != 1 {
			t.Fatalf(`Log message "%v" found %v times, expected exactly 1 time`, uniqueLog, timesFound)
		}
	}
}

func TestVerify_WithLocalArtifact(t *testing.T) {
	tests := []struct {
		description     string
		dir             string
		profile         string
		shouldErr       bool
		shouldBuild     bool
		expectedMsgs    []string
		notExpectedMsgs []string
	}{
		{
			description: "build and verify",
			dir:         "testdata/verify-succeed",
			profile:     "local-built-artifact",
			shouldBuild: true,
			expectedMsgs: []string{
				"Tags used in verification:",
				"- localtask ->",
				"[localtask] Hello world ! 0",
				"[alpine-1] alpine-1",
			},
			notExpectedMsgs: []string{
				"- img-not-used-in-verify ->",
			},
		},
		{
			description: "fail due not found image",
			dir:         "testdata/verify-succeed-k8s",
			profile:     "local-built-artifact",
			shouldErr:   true,
			expectedMsgs: []string{
				"1 error(s) occurred",
				"creating container for localtask: ErrImagePull",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)

			ns, _ := SetupNamespace(t.T)

			args := []string{"-p", test.profile}

			if test.shouldBuild {
				tmpfile := testutil.TempFile(t.T, "", []byte{})
				skaffold.Build(append(args, "--file-output", tmpfile)...).InDir(test.dir).RunOrFail(t.T)
				args = append(args, "--build-artifacts", tmpfile)
			}

			out, err := skaffold.Verify(args...).InDir(test.dir).InNs(ns.Name).RunWithCombinedOutput(t.T)
			logs := string(out)

			t.CheckError(test.shouldErr, err)

			for _, expectedMsg := range test.expectedMsgs {
				t.CheckContains(expectedMsg, logs)
			}

			for _, notExpectedMsg := range test.notExpectedMsgs {
				testutil.CheckNotContains(t.T, notExpectedMsg, logs)
			}
		})
	}
}
