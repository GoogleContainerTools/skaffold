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
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCacheAPITriggers(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	// Run skaffold build first to cache artifacts.
	skaffold.Build().InDir("examples/getting-started").RunOrFail(t)

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	rpcAddr := randomPort()

	stop := skaffold.Dev("--rpc-port", rpcAddr).InDir("examples/getting-started").InNs(ns.Name).RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	stream, err := readEventAPIStream(client, t, readRetries)
	if stream == nil {
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// read entries from the log
	entries := make(chan *proto.LogEntry)
	go func() {
		for {
			entry, _ := stream.Recv()
			if entry != nil {
				entries <- entry
			}
		}
	}()

	// Ensure we see a build triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetBuildEvent().GetArtifact() == "gcr.io/k8s-skaffold/skaffold-example", nil
	})
	testutil.CheckError(t, false, err)
}
