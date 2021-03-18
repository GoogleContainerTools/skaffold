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

	expectedText := "bar\n"
	testDir := "testdata/test-events"
	testFile := "testdata/test-events/test"
	defer func() {
		defer os.Remove(testFile)
	}()

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir(testDir).RunOrFail(t)

	ns, client := SetupNamespace(t)
	rpcAddr := randomPort()

	skaffold.Dev("--rpc-port", rpcAddr).InDir(testDir).InNs(ns.Name).RunBackground(t)

	client.WaitForPodsReady("test-events-example")

	// Ensure we see a test is triggered in the event log
	_, entries := apiEvents(t, rpcAddr)

	waitForTestEvent(t, entries, func(e *proto.LogEntry) bool {
		return (e.GetEvent().GetTestEvent().GetStatus() != InProgress)
	})

	verifyTestCompletedWithEvents(t, entries, expectedText, testFile)
}

func waitForTestEvent(t *testing.T, entries chan *proto.LogEntry, condition func(*proto.LogEntry) bool) {
	failNowIfError(t, wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) { return condition(<-entries), nil }))
}

func verifyTestCompletedWithEvents(t *testing.T, entries chan *proto.LogEntry, expectedText string, fileName string) {
	// Ensure we see a file sync in progress triggered in the event log
	err := wait.Poll(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		event := e.GetEvent().GetTestEvent()
		return event != nil && event.GetStatus() == InProgress, nil
	})
	failNowIfError(t, err)

	err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, e := ioutil.ReadFile(fileName)
		failNowIfError(t, e)
		return string(out) == expectedText, nil
	})
	failNowIfError(t, err)
}
