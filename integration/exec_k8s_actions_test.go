/*
Copyright 2023 The Skaffold Authors

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
	"testing"

	"github.com/google/uuid"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestExec_K8SActions(t *testing.T) {
	tests := []struct {
		description     string
		action          string
		shouldErr       bool
		envFile         string
		expectedMsgs    []string
		notExpectedLogs []string
	}{
		{
			description:     "fail due to action timeout",
			action:          "action-fail-timeout",
			shouldErr:       true,
			expectedMsgs:    []string{"context deadline exceeded"},
			notExpectedLogs: []string{"[task1] bye-from-task1"},
		},
		{
			description:     "fail with fail fast",
			action:          "action-fail-fast",
			shouldErr:       true,
			expectedMsgs:    []string{`error in task4 job execution, job failed`},
			notExpectedLogs: []string{"[task3] bye-from-task3"},
		},
		{
			description: "fail with fail safe",
			action:      "action-fail-safe-logs",
			shouldErr:   true,
			expectedMsgs: []string{
				"[task5l] hello-from-task5l",
				"[task5l] bye-from-task5l",
				`* error in task6l job execution, job failed`,
			},
		},
		{
			description: "action succeeded",
			action:      "action-succeeded-logs",
			envFile:     "exec.env",
			expectedMsgs: []string{
				"[task7l] hello-from-env-file",
				"[task7l] bye-from-env-file",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)
			args := []string{test.action}

			if test.envFile != "" {
				args = append(args, "--env-file", test.envFile)
			}

			out, err := skaffold.Exec(args...).InDir("testdata/custom-actions-k8s").RunWithCombinedOutput(t.T)
			t.CheckError(test.shouldErr, err)
			logs := string(out)

			for _, expectedMsg := range test.expectedMsgs {
				t.CheckContains(expectedMsg, logs)
			}

			for _, nel := range test.notExpectedLogs {
				testutil.CheckNotContains(t.T, nel, logs)
			}
		})
	}
}

func TestExec_K8SActionWithLocalArtifact(t *testing.T) {
	tests := []struct {
		description  string
		action       string
		shouldErr    bool
		shouldBuild  bool
		expectedMsgs []string
	}{
		{
			description: "fail due not found image",
			action:      "action-with-local-built-img-1",
			shouldErr:   true,
			expectedMsgs: []string{
				"creating container for local-img-task1-1: ErrImagePull",
			},
		},
		{
			description: "build and run task",
			action:      "action-with-local-built-img-2",
			shouldBuild: true,
			expectedMsgs: []string{
				"[local-img-task1-2] Hello world from-local-img! 4",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)
			dir := "testdata/custom-actions-k8s"
			args := []string{test.action}

			if test.shouldBuild {
				tmpfile := testutil.TempFile(t.T, "", []byte{})
				skaffold.Build("--file-output", tmpfile, "--tag", uuid.New().String(), "--check-cluster-node-platforms=true").InDir(dir).RunOrFail(t.T)
				args = append(args, "--build-artifacts", tmpfile)
			}

			out, err := skaffold.Exec(args...).InDir(dir).RunWithCombinedOutput(t.T)
			t.CheckError(test.shouldErr, err)

			for _, expectedMsg := range test.expectedMsgs {
				t.CheckContains(expectedMsg, string(out))
			}
		})
	}
}

func TestExec_K8SActionsEvents(t *testing.T) {
	tests := []struct {
		description  string
		action       string
		shouldErr    bool
		expectedLogs []string
	}{
		{
			description: "events for succeeded action",
			action:      "action-succeeded",
			expectedLogs: []string{
				`"taskEvent":{"id":"Exec-0","task":"Exec","description":"Executing custom action action-succeeded","status":"InProgress"}}`,
				`"execEvent":{"id":"task7","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task8","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task7","taskId":"Exec-0","status":"Succeeded"}}`,
				`"execEvent":{"id":"task8","taskId":"Exec-0","status":"Succeeded"}}`,
				// TODO(#8728): Uncomment this expected log line when the flaky behaviour is solved.
				// `"taskEvent":{"id":"Exec-0","task":"Exec","status":"Succeeded"}}`,
			},
		},
		{
			description: "events for fail action - fail fast",
			action:      "action-fail-fast",
			shouldErr:   true,
			expectedLogs: []string{
				`"taskEvent":{"id":"Exec-0","task":"Exec","description":"Executing custom action action-fail-fast","status":"InProgress"}}`,
				`"execEvent":{"id":"task3","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task4","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task4","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"error in task4 job execution, job failed"`,
				`"execEvent":{"id":"task3","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"error in task3 job execution, event type: ERROR"`,
				`"taskEvent":{"id":"Exec-0","task":"Exec","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"error in task4 job execution, job failed"`,
			},
		},
		{
			description: "events for fail action - fail safe",
			action:      "action-fail-safe",
			shouldErr:   true,
			expectedLogs: []string{
				`"taskEvent":{"id":"Exec-0","task":"Exec","description":"Executing custom action action-fail-safe","status":"InProgress"}}`,
				`"execEvent":{"id":"task5","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task6","taskId":"Exec-0","status":"InProgress"}}`,
				`"execEvent":{"id":"task6","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"error in task6 job execution, job failed"`,
				`"execEvent":{"id":"task5","taskId":"Exec-0","status":"Succeeded"}}`,
				`"taskEvent":{"id":"Exec-0","task":"Exec","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"1 error(s) occurred:\n* error in task6 job execution, job failed"`,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)
			rpcAddr := randomPort()
			tmp := t.TempDir()
			logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

			args := []string{test.action, "--rpc-port", rpcAddr, "--event-log-file", logFile}

			_, err := skaffold.Exec(args...).InDir("testdata/custom-actions-k8s").RunWithCombinedOutput(t.T)

			t.CheckError(test.shouldErr, err)

			b, err := os.ReadFile(logFile + ".v2")
			if err != nil {
				t.Fatalf("error reading %s", logFile+".v2")
			}
			v2EventLogs := string(b)
			for _, expectedLog := range test.expectedLogs {
				t.CheckContains(expectedLog, v2EventLogs)
			}
		})
	}
}
