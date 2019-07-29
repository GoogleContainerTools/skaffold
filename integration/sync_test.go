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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDevSync(t *testing.T) {
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
			if testing.Short() {
				t.Skip("skipping integration test")
			}
			if ShouldRunGCPOnlyTests() {
				t.Skip("skipping test that is not gcp only")
			}

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, k8sclient, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Build().InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunOrFail(t)

	rpcAddr := randomPort()
	client, shutdown := setupRPCClient(t, rpcAddr)
	defer shutdown()

	stop := skaffold.Dev("--auto-sync=false", "--rpc-port", rpcAddr).InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunBackground(t)
	defer stop()

	k8sclient.WaitForPodsReady("test-file-sync")

	ioutil.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	client.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Sync: true,
		},
	})

	err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", "foo").Output()
		return string(out) == "foo", nil
	})
	testutil.CheckError(t, false, err)
}
