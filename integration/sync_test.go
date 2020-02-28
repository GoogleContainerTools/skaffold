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
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
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
			failNowIfError(t, err)
		})
	}
}

func TestDevAutoSync(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	dir := "testdata/jib-sync/"

	tests := []struct {
		description string
		profiles    []string
		uniqueStr   string
	}{
		{
			description: "jib maven auto sync",
			profiles:    []string{"maven"},
			uniqueStr:   "maven-maven",
		},
		{
			description: "jib gradle auto sync",
			profiles:    []string{"gradle"},
			uniqueStr:   "gradle-gradle",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().WithRepo("gcr.io/appu-learn").WithProfiles(test.profiles).InDir(dir).RunOrFail(t)

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Dev("--trigger", "notify").WithRepo("gcr.io/appu-learn").WithProfiles(test.profiles).InDir(dir).InNs(ns.Name).RunBackground(t)
			defer stop()

			client.WaitForPodsReady("test-file-sync")

			// direct file sync (this file is an existing file checked in for this testdata)
			directFile := "direct-file"
			directFilePath := dir + "src/main/jib/" + directFile
			directFileData := "direct-data"
			if err := ioutil.WriteFile(directFilePath, []byte(directFileData), 0644); err != nil {
				t.Fatalf("Failed to write local file to sync %s", directFilePath)
			}
			defer func() { os.Truncate(directFilePath, 0) }()

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", directFile).Output()
				return string(out) == directFileData, nil
			})
			failNowIfError(t, err)

			// compile and sync
			generatedFileSrc := dir + "src/main/java/hello/HelloController.java"
			if oldContents, err := ioutil.ReadFile(generatedFileSrc); err != nil {
				t.Fatalf("Failed to read file %s", generatedFileSrc)
			} else {
				newContents := strings.Replace(string(oldContents), "text-to-replace", test.uniqueStr, 1)
				if err := ioutil.WriteFile(generatedFileSrc, []byte(newContents), 0644); err != nil {
					t.Fatalf("Failed to write new contents to file %s", generatedFileSrc)
				}
				defer func() { ioutil.WriteFile(generatedFileSrc, oldContents, 0644) }()
			}
			err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				// distroless debug only has wget, not curl
				out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "wget", "localhost:8080/", "-q", "-O", "-").Output()
				return string(out) == test.uniqueStr, nil
			})
			failNowIfError(t, err)
		})
	}
}

func TestDevSyncAPITrigger(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Build().InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunOrFail(t)

	rpcAddr := randomPort()

	stop := skaffold.Dev("--auto-sync=false", "--rpc-port", rpcAddr).InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunBackground(t)
	defer stop()

	rpcClient, entries, shutdown := apiEvents(t, rpcAddr)
	defer shutdown()

	// throw away first 5 entries of log (from first run of dev loop)
	for i := 0; i < 5; i++ {
		<-entries
	}

	client.WaitForPodsReady("test-file-sync")

	ioutil.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Sync: true,
		},
	})

	// Ensure we see a file sync in progress triggered in the event log
	err := wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetFileSyncEvent().GetStatus() == "In Progress", nil
	})
	failNowIfError(t, err)

	err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", "foo").Output()
		return string(out) == "foo", nil
	})
	failNowIfError(t, err)

	// Ensure we see a file sync succeeded triggered in the event log
	err = wait.PollImmediate(time.Millisecond*500, 2*time.Minute, func() (bool, error) {
		e := <-entries
		return e.GetEvent().GetFileSyncEvent().GetStatus() == "Succeeded", nil
	})
	failNowIfError(t, err)
}
