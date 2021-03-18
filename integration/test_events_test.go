/*
Copyright 2021 The Skaffold Authors

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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

func TestTestEvents(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		podName      string
		expectedText string
		testDir      string
		testFile     string
		config       string
		testType     string
	}{
		{
			description:  "test events for custom test",
			podName:      "custom-test-events",
			expectedText: "bar\n",
			testDir:      "testdata/test-events/custom",
			testFile:     "testdata/test-events/custom/test",
			config:       "skaffold.yaml",
			testType:     "custom",
		},
		{
			description: "test events for structure test",
			podName:     "structure-test-events",
			testDir:     "testdata/test-events/structure",
			testFile:    "testdata/test-events/test",
			config:      "skaffold.yaml",
			testType:    "structure",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func() {
				defer os.Remove(test.testFile)
			}()

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir(test.testDir).WithConfig(test.config).RunOrFail(t)

			ns, client := SetupNamespace(t)
			rpcAddr := randomPort()

			skaffold.Dev("--rpc-port", rpcAddr).InDir(test.testDir).WithConfig(test.config).InNs(ns.Name).RunBackground(t)

			client.WaitForPodsReady(test.podName)

			// Ensure we see a test is triggered in the event log
			_, entries := apiEvents(t, rpcAddr)

			waitForTestEvent(t, entries, func(e *proto.LogEntry) bool {
				return (e.GetEvent().GetTestEvent().GetStatus() != InProgress)
			})

			verifyTestCompletedWithEvents(t, entries, test.testType, test.expectedText, test.testFile)
		})
	}
}

func waitForTestEvent(t *testing.T, entries chan *proto.LogEntry, condition func(*proto.LogEntry) bool) {
	failNowIfError(t, wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) { return condition(<-entries), nil }))
}

func verifyTestCompletedWithEvents(t *testing.T, entries chan *proto.LogEntry, testType string, expectedText string, fileName string) {
	// Ensure we see a test in progress triggered in the event log
	err := wait.Poll(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		event := e.GetEvent().GetTestEvent()
		return event != nil && event.GetStatus() == InProgress, nil
	})
	failNowIfError(t, err)

	switch testType {
	case "Custom":
		err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
			out, e := ioutil.ReadFile(fileName)
			failNowIfError(t, e)
			return string(out) == expectedText, nil
		})
		failNowIfError(t, err)
	default:
		break
	}
}
