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

func TestExec_LocalActions(t *testing.T) {
	tests := []struct {
		description  string
		action       string
		shouldErr    bool
		envFile      string
		expectedMsgs []string
	}{

		{
			description:  "fail due to action timeout",
			action:       "action-fail-timeout",
			shouldErr:    true,
			expectedMsgs: []string{"context deadline exceeded"},
		},
		{
			description:  "fail with fail fast",
			action:       "action-fail-fast",
			shouldErr:    true,
			expectedMsgs: []string{`"task4" running container image "alpine:3.15.4" errored during run with status code: 1`},
		},
		{
			description: "fail with fail safe",
			action:      "action-fail-safe",
			shouldErr:   true,
			expectedMsgs: []string{
				"[task5] hello-from-task5",
				"[task5] bye-from-task5",
				`* "task6" running container image "alpine:3.15.4" errored during run with status code: 1`,
			},
		},
		{
			description: "action succeeded",
			action:      "action-succeeded",
			envFile:     "exec.env",
			expectedMsgs: []string{
				"[task7] hello-from-env-file",
				"[task7] bye-from-env-file",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			args := []string{test.action}

			if test.envFile != "" {
				args = append(args, "--env-file", test.envFile)
			}

			out, err := skaffold.Exec(args...).InDir("testdata/custom-actions-local").RunWithCombinedOutput(t.T)
			t.CheckError(test.shouldErr, err)

			for _, expectedMsg := range test.expectedMsgs {
				t.CheckContains(expectedMsg, string(out))
			}
		})
	}
}

func TestExec_LocalActionWithLocalArtifact(t *testing.T) {
	tests := []struct {
		description  string
		action       string
		shouldErr    bool
		shouldBuild  bool
		expectedMsgs []string
	}{
		{
			description: "fail due not found image",
			action:      "action-with-local-built-img",
			shouldErr:   true,
			expectedMsgs: []string{
				"Error response from daemon: pull access denied for localtaks",
			},
		},
		{
			description: "build and run task",
			action:      "action-with-local-built-img",
			shouldBuild: true,
			expectedMsgs: []string{
				"[local-img-task1] Hello world from-local-img! 4",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			dir := "testdata/custom-actions-local"
			args := []string{test.action}

			if test.shouldBuild {
				tmpfile := testutil.TempFile(t.T, "", []byte{})
				skaffold.Build("--file-output", tmpfile).InDir(dir).RunOrFail(t.T)
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

func TestExec_ActionsEvents(t *testing.T) {
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
				`"execEvent":{"id":"task4","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"\"task4\" running container image \"alpine:3.15.4\" errored during run with status code: 1"`,
				`"execEvent":{"id":"task3","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"context canceled"`,
				`"taskEvent":{"id":"Exec-0","task":"Exec","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"\"task4\" running container image \"alpine:3.15.4\" errored during run with status code: 1"`,
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
				`"execEvent":{"id":"task6","taskId":"Exec-0","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"\"task6\" running container image \"alpine:3.15.4\" errored during run with status code: 1"`,
				`"execEvent":{"id":"task5","taskId":"Exec-0","status":"Succeeded"}}`,
				`"taskEvent":{"id":"Exec-0","task":"Exec","status":"Failed","actionableErr":{"errCode":"UNKNOWN_ERROR","message":"1 error(s) occurred:\n* \"task6\" running container image \"alpine:3.15.4\" errored during run with status code: 1"`,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			rpcAddr := randomPort()
			tmp := t.TempDir()
			logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

			args := []string{test.action, "--rpc-port", rpcAddr, "--event-log-file", logFile}

			_, err := skaffold.Exec(args...).InDir("testdata/custom-actions-local").RunWithCombinedOutput(t.T)

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
