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
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	testDev = "test-dev"
)

func TestControlAPIManualTriggers(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunWithCombinedOutput(t)

	rpcClient, entries := apiEvents(t, rpcAddr)

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		<-entries
	}

	dep := client.GetDeployment("test-dev")

	// Make a change to foo
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

	// Execute dev loop trigger
	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Devloop: true,
		},
	})
	// Ensure we see a build triggered in the event log
	err := wait.Poll(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetBuildEvent().GetArtifact() == testDev, nil
	})
	failNowIfError(t, err)
	// verify deployment happened.
	verifyDeployment(t, entries, client, dep)

	// Make another change to foo and we should not see any event log.
	Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")
	// Ensure we dont see a build triggered in the event log.
	err = wait.Poll(time.Millisecond*500, 1*time.Second, func() (bool, error) {
		e := <-entries
		return e.GetEvent() != nil, nil
	})
	if err == nil {
		t.Error("expected no events saw one.")
	}

}
