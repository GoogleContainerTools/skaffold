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
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	yamlpatch "github.com/krishicks/yaml-patch"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	event "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	V2proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
)

// TODO: remove nolint once we've reenabled integration tests
//
//nolint:golint,unused
var syncTests = []struct {
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

func TestDevSync(t *testing.T) {
	for _, test := range syncTests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			ns, client := SetupNamespace(t)

			rpcAddr := randomPort()

			skaffold.Dev("--rpc-port", rpcAddr, "--trigger", test.trigger).InDir("testdata/file-sync").WithConfig(test.config).InNs(ns.Name).RunBackground(t)

			client.WaitForPodsReady("test-file-sync")

			_, entries := v2apiEvents(t, rpcAddr)

			failNowIfError(t, waitForV2Event(90*time.Second, entries, func(e *V2proto.Event) bool {
				taskEvent, ok := e.EventType.(*V2proto.Event_TaskEvent)
				return ok && taskEvent.TaskEvent.Task == string(constants.DevLoop) && taskEvent.TaskEvent.Status == event.Succeeded
			}))

			os.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
			defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", ns.Name, "--", "cat", "foo").Output()
				return string(out) == "foo", nil
			})
			failNowIfError(t, err)
		})
	}
}

func TestDevSyncDefaultNamespace(t *testing.T) {
	for _, test := range syncTests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			manifest, err := os.ReadFile("testdata/file-sync/pod.yaml")
			defer os.WriteFile("testdata/file-sync/pod.yaml", manifest, 0644)
			if err != nil {
				t.Fatal("Failed to read from file-sync/pod.yaml file.")
			}
			id := "test-file-sync-" + uuid.New().String()
			ops := []byte(
				`---
- op: replace
  path: /metadata/name
  value: ` + id)

			patch, err := yamlpatch.DecodePatch(ops)
			if err != nil {
				t.Fatal(err)
			}

			nm, err := patch.Apply(manifest)
			if err != nil {
				t.Fatal(err)
			}
			os.WriteFile("testdata/file-sync/pod.yaml", nm, 0644)

			_, client := DefaultNamespace(t)

			rpcAddr := randomPort()

			skaffold.Dev("--rpc-port", rpcAddr, "--trigger", test.trigger).InDir("testdata/file-sync").WithConfig(test.config).RunBackground(t)

			defer skaffold.Delete().InDir("testdata/file-sync").WithConfig(test.config).Run(t)

			client.WaitForPodsReady(id)

			_, entries := v2apiEvents(t, rpcAddr)
			failNowIfError(t, waitForV2Event(90*time.Second, entries, func(e *V2proto.Event) bool {
				taskEvent, ok := e.EventType.(*V2proto.Event_TaskEvent)
				return ok && taskEvent.TaskEvent.Task == string(constants.DevLoop) && taskEvent.TaskEvent.Status == event.Succeeded
			}))

			os.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
			defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

			err = wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				out, _ := exec.Command("kubectl", "exec", id, "--", "cat", "foo").Output()
				return string(out) == "foo", nil
			})
			failNowIfError(t, err)
		})
	}
}

func TestDevAutoSync(t *testing.T) {
	dir := "examples/jib-sync/"

	tests := []struct {
		description string
		configFile  string
		profiles    []string
		uniqueStr   string
	}{
		{
			description: "jib maven auto sync",
			configFile:  "skaffold-maven.yaml",
			uniqueStr:   "maven-maven",
		},
		{
			description: "jib gradle auto sync",
			configFile:  "skaffold-gradle.yaml",
			uniqueStr:   "gradle-gradle",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			ns, client := SetupNamespace(t)

			rpcAddr := randomPort()
			output := skaffold.Dev("--trigger", "notify", "--rpc-port", rpcAddr).WithConfig(test.configFile).InDir(dir).InNs(ns.Name).RunLive(t)

			client.WaitForPodsReady("test-file-sync")

			// give the server a chance to warm up, this integration test on slow environments (KIND on travis)
			// fails because of a potential server race condition.
			scanner := bufio.NewScanner(output)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "Started Application") {
					err := output.Close()
					if err != nil {
						t.Fatal("failed to close skaffold dev output reader during test")
					}
					return
				}
			}

			_, entries := v2apiEvents(t, rpcAddr)

			failNowIfError(t, waitForV2Event(90*time.Second, entries, func(e *V2proto.Event) bool {
				taskEvent, ok := e.EventType.(*V2proto.Event_TaskEvent)
				return ok && taskEvent.TaskEvent.Task == string(constants.DevLoop) && taskEvent.TaskEvent.Status == event.Succeeded
			}))

			// direct file sync (this file is an existing file checked in for this testdata)
			directFile := "direct-file"
			directFilePath := dir + "src/main/jib/" + directFile
			directFileData := "direct-data"
			if err := os.WriteFile(directFilePath, []byte(directFileData), 0644); err != nil {
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
			if oldContents, err := os.ReadFile(generatedFileSrc); err != nil {
				t.Fatalf("Failed to read file %s", generatedFileSrc)
			} else {
				newContents := strings.Replace(string(oldContents), "text-to-replace", test.uniqueStr, 1)
				if err := os.WriteFile(generatedFileSrc, []byte(newContents), 0644); err != nil {
					t.Fatalf("Failed to write new contents to file %s", generatedFileSrc)
				}
				defer func() {
					os.WriteFile(generatedFileSrc, oldContents, 0644)
				}()
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
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	skaffold.Build().InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunOrFail(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-sync=false", "--rpc-port", rpcAddr).InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)
	client.WaitForPodsReady("test-file-sync")
	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

	os.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	rpcClient.Execute(context.Background(), &proto.UserIntentRequest{
		Intent: &proto.Intent{
			Sync: true,
		},
	})

	verifySyncCompletedWithEvents(t, entries, ns.Name, "foo")
}

func TestDevAutoSyncAPITrigger(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, client := SetupNamespace(t)

	skaffold.Build().InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunOrFail(t)

	rpcAddr := randomPort()
	skaffold.Dev("--auto-sync=false", "--rpc-port", rpcAddr).InDir("testdata/file-sync").WithConfig("skaffold-manual.yaml").InNs(ns.Name).RunBackground(t)

	rpcClient, entries := apiEvents(t, rpcAddr)

	for i := 0; i < 5; i++ {
		<-entries
	}

	client.WaitForPodsReady("test-file-sync")

	failNowIfError(t, waitForEvent(90*time.Second, entries, func(e *proto.LogEntry) bool {
		dle, ok := e.Event.EventType.(*proto.Event_DevLoopEvent)
		return ok && dle.DevLoopEvent.Status == event.Succeeded
	}))

	os.WriteFile("testdata/file-sync/foo", []byte("foo"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	rpcClient.AutoSync(context.Background(), &proto.TriggerRequest{
		State: &proto.TriggerState{
			Val: &proto.TriggerState_Enabled{
				Enabled: true,
			},
		},
	})

	verifySyncCompletedWithEvents(t, entries, ns.Name, "foo")

	os.WriteFile("testdata/file-sync/foo", []byte("bar"), 0644)
	defer func() { os.Truncate("testdata/file-sync/foo", 0) }()

	verifySyncCompletedWithEvents(t, entries, ns.Name, "bar")

	rpcClient.AutoSync(context.Background(), &proto.TriggerRequest{
		State: &proto.TriggerState{
			Val: &proto.TriggerState_Enabled{
				Enabled: true,
			},
		},
	})
}

func verifySyncCompletedWithEvents(t *testing.T, entries chan *proto.LogEntry, namespace string, fileContent string) {
	// Ensure we see a file sync in progress triggered in the event log
	err := waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		event := e.GetEvent().GetFileSyncEvent()
		return event != nil && event.GetStatus() == InProgress
	})
	failNowIfError(t, err)

	err = wait.Poll(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
		out, _ := exec.Command("kubectl", "exec", "test-file-sync", "-n", namespace, "--", "cat", "foo").Output()
		return string(out) == fileContent, nil
	})
	failNowIfError(t, err)

	// Ensure we see a file sync succeeded triggered in the event log
	err = waitForEvent(2*time.Minute, entries, func(e *proto.LogEntry) bool {
		event := e.GetEvent().GetFileSyncEvent()
		return event != nil && event.GetStatus() == "Succeeded"
	})
	failNowIfError(t, err)
}
