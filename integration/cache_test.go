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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

func TestCacheAPITriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("examples/getting-started").RunOrFail(t)

	ns, client := SetupNamespace(t)
	rpcAddr := randomPort()

	// Disable caching to ensure we get a "build in progress" event each time.
	skaffold.Dev("--cache-artifacts=false", "--rpc-port", rpcAddr).InDir("examples/getting-started").InNs(ns.Name).RunBackground(t)
	client.WaitForPodsReady("getting-started")
	// Ensure we see a build triggered in the event log
	_, entries := apiEvents(t, rpcAddr)

	failNowIfError(t, waitForEvent(1*time.Minute, entries, func(e *proto.LogEntry) bool {
		return e.GetEvent().GetBuildEvent().GetArtifact() == "skaffold-example"
	}))
}

func waitForEvent(timeout time.Duration, entries chan *proto.LogEntry, condition func(*proto.LogEntry) bool) error {
	ctx, cancelTimeout := context.WithTimeout(context.Background(), timeout)
	defer cancelTimeout()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for condition on log entry")
		case ev := <-entries:
			if condition(ev) {
				return nil
			}
		}
	}
}
