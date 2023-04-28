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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

func TestTestEvents(t *testing.T) {
	tests := []struct {
		description string
		podName     string
		testDir     string
		config      string
		args        []string
		numOfTests  int
	}{
		{
			description: "test events for custom test",
			podName:     "test-events",
			testDir:     "testdata/test-events",
			config:      "skaffold.yaml",
			args:        []string{"--profile", "custom"},
			numOfTests:  1,
		},
		{
			description: "test events for structure test",
			podName:     "test-events",
			testDir:     "testdata/test-events",
			config:      "skaffold.yaml",
			args:        []string{"--profile", "structure"},
			numOfTests:  1,
		},
		{
			description: "test events for custom & structure tests",
			podName:     "test-events",
			testDir:     "testdata/test-events",
			config:      "skaffold.yaml",
			args:        []string{"--profile", "customandstructure"},
			numOfTests:  2,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build(test.args...).InDir(test.testDir).WithConfig(test.config).RunOrFail(t)

			ns, client := SetupNamespace(t)
			rpcAddr := randomPort()

			// test.args...
			args := append(test.args, "--rpc-port", rpcAddr)
			skaffold.Dev(args...).InDir(test.testDir).WithConfig(test.config).InNs(ns.Name).RunLive(t)

			client.WaitForPodsReady(test.podName)

			// Ensure we see a test is triggered in the event log
			_, entries := apiEvents(t, rpcAddr)

			for i := 0; i < test.numOfTests; i++ {
				verifyTestCompletedWithEvents(t, entries)
			}
		})
	}
}

func verifyTestCompletedWithEvents(t *testing.T, entries chan *proto.LogEntry) {
	// Ensure we see a test in progress triggered in the event log
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		event := e.GetEvent().GetTestEvent()
		return event != nil && event.GetStatus() == InProgress
	})
	failNowIfError(t, err)

	// Ensure we see the test completed triggered in the event log
	err = waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		event := e.GetEvent().GetTestEvent()
		return event != nil && event.GetStatus() == "Complete"
	})
	failNowIfError(t, err)
}
