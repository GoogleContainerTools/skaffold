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

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	testDev = "test-dev"
)

func TestControlAPIManualTriggers(t *testing.T) {
	// TODO: https://github.com/GoogleContainerTools/skaffold/issues/7029
	t.Skipf("TODO Fix: https://github.com/GoogleContainerTools/skaffold/issues/7029")
	MarkIntegrationTest(t, CanRunWithoutGcp)

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	ns, client := SetupNamespace(t)
	out := bytes.Buffer{}
	rpcAddr := randomPort()
	skaffold.Dev("--auto-build=false", "--auto-sync=false", "--auto-deploy=false", "--rpc-port", rpcAddr, "--cache-artifacts=false").InDir("testdata/dev").InNs(ns.Name).RunInBackgroundWithOutput(t, &out)

	rpcClient, entries := apiEvents(t, rpcAddr)

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		<-entries
	}

	dep := client.GetDeployment(testDev)

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

	// Give skaffold some time to register a file change.
	time.Sleep(1 * time.Second)
	if c := strings.Count(out.String(), "Generating tags"); c != 2 {
		failNowIfError(t, fmt.Errorf("expected to see tags generated twice (1st build and 1 trigger), saw %d times", c))
	}
}
