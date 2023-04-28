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

	rpcPort := randomPort2()
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

	rpcPort := randomPort2()
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

	rpcPort := randomPort2()
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

	rpcPort := randomPort2()
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

func checkUniqueLogs(t *testutil.T, logs string, expectedUniqueLogs []string) {
	for _, uniqueLog := range expectedUniqueLogs {
		timesFound := strings.Count(logs, uniqueLog)
		if timesFound != 1 {
			t.Fatalf(`Log message "%v" found %v times, expected exactly 1 time`, uniqueLog, timesFound)
		}
	}
}
