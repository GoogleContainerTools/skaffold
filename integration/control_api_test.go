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
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	event "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

const (
	testDev = "test-dev"
)

func TestControlAPIManualTriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	ns, client := SetupNamespace(t)
	out := bytes.Buffer{}
	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunInBackgroundWithOutput(t, &out)

	rpcClient, entries := apiEvents(t, rpcAddr)

	dep := client.GetDeployment(testDev)

	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Execute dev loop trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Devloop: true,
		},
	})
	// Ensure we see a build triggered in the event log
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetBuildEvent().GetArtifact() == testDev
	})
	failNowIfError(t, err)
	// verify deployment happened.
	verifyDeployment(t, entries, client, dep)

	// Make another change to foo and we should not see any event log.
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Give skaffold some time to register a file change.
	time.Sleep(1 * time.Second)
	if c := strings.Count(out.String(), "Generating tags"); c != 2 {
		failNowIfError(t, fmt.Errorf("expected to see tags generated twice (1st build and 1 trigger), saw %d times", c))
	}
}
