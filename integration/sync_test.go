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
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDevSync(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	tests := []struct {
		description string
		trigger     string
		config      string
	}{
		{
			description: "manual sync with polling trigger",
			trigger:     "polling",
			config:      "skaffold-manual.yaml",
		},
		{
			description: "manual sync with notify trigger",
			trigger:     "notify",
			config:      "skaffold-manual.yaml",
		},
		{
			description: "inferred sync with notify trigger",
			trigger:     "notify",
			config:      "skaffold-infer.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir("testdata/file-sync").WithConfig(test.config).RunOrFail(t)

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Dev("--trigger", test.trigger).InDir("testdata/file-sync").WithConfig(test.config).InNs(ns.Name).RunBackground(t)
			defer stop()

			client.WaitForPodsReady("test-file-sync")

			ioutil.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
			defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", "foo").Output()
				return string(out) == "foo", nil
			})
			testutil.CheckError(t, false, err)
		})
	}
}

func TestDevSyncAPITrigger(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	ns, k8sclient, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Build().InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunOrFail(t)

	rpcAddr := randomPort()

	stop := skaffold.Dev("--auto-sync=false", "--rpc-port", rpcAddr).InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunBackground(t)
	defer stop()

	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	stream, err := readEventAPIStream(client, t, readRetries)
	if stream == nil {
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		stream.Recv()
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

	k8sclient.WaitForPodsReady("test-file-sync")

	ioutil.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	client.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Sync: true,
		},
	})

	// Ensure we see a file sync in progress triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetFileSyncEvent().GetStatus() == "In Progress", nil
	})
	testutil.CheckError(t, false, err)

	err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", "foo").Output()
		return string(out) == "foo", nil
	})
	testutil.CheckError(t, false, err)

	// Ensure we see a file sync succeeded triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetFileSyncEvent().GetStatus() == "Succeeded", nil
	})
	testutil.CheckError(t, false, err)
}
