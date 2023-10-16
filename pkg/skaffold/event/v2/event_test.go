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

package v2

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var targetPort = proto.IntOrString{Type: 0, IntVal: 2001}

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 1000; step++ {
		ev := newHandler()

		ev.logEvent(&proto.Event{
			EventType: &proto.Event_SkaffoldLogEvent{
				SkaffoldLogEvent: &proto.SkaffoldLogEvent{Message: "OLD"},
			},
		})
		go func() {
			ev.logEvent(&proto.Event{
				EventType: &proto.Event_SkaffoldLogEvent{
					SkaffoldLogEvent: &proto.SkaffoldLogEvent{Message: "FRESH"},
				},
			})
			ev.logEvent(&proto.Event{
				EventType: &proto.Event_SkaffoldLogEvent{
					SkaffoldLogEvent: &proto.SkaffoldLogEvent{Message: "POISON PILL"},
				},
			})
		}()

		var received int32
		ev.forEachEvent(func(e *proto.Event) error {
			if e.GetSkaffoldLogEvent().Message == "POISON PILL" {
				return errors.New("Done")
			}

			atomic.AddInt32(&received, 1)
			return nil
		})

		if atomic.LoadInt32(&received) != 2 {
			t.Fatalf("Expected %d events, Got %d (Step: %d)", 2, received, step)
		}
	}
}

func wait(t *testing.T, condition func() bool) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}

		case <-timeout.C:
			t.Fatal("Timed out waiting")
		}
	}
}

func TestSaveEventsToFile(t *testing.T) {
	// Generate the file name to dump the file. File and directory should be created if it doesn't exist.
	fName := filepath.Join(os.TempDir(), "test", "logfile")
	t.Cleanup(func() { os.RemoveAll(fName) })
	// add some events to the event log
	handler.eventLog = []*proto.Event{
		{
			EventType: &proto.Event_BuildSubtaskEvent{},
		}, {
			EventType: &proto.Event_TaskEvent{},
		},
	}

	// save events to file
	if err := SaveEventsToFile(fName); err != nil {
		t.Fatalf("error saving events to file: %v", err)
	}

	extractInfoFromFile := func(fName string) (int, int, int) {
		// ensure that the events in the file match the event log
		contents, err := os.ReadFile(fName)
		if err != nil {
			t.Fatalf("reading tmp file: %v", err)
		}

		var logEntries []*proto.Event
		entries := strings.Split(string(contents), "\n")
		for _, e := range entries {
			if e == "" {
				continue
			}
			var logEntry proto.Event
			if err := jsonpb.UnmarshalString(e, &logEntry); err != nil {
				t.Errorf("error converting http response %s to proto: %s", e, err.Error())
			}
			logEntries = append(logEntries, &logEntry)
		}

		buildCompleteEvent, devLoopCompleteEvent := 0, 0
		for _, entry := range logEntries {
			t.Log(entry.GetEventType())
			switch entry.GetEventType().(type) {
			case *proto.Event_BuildSubtaskEvent:
				buildCompleteEvent++
				t.Logf("build event %d: %v", buildCompleteEvent, entry)
			case *proto.Event_TaskEvent:
				devLoopCompleteEvent++
				t.Logf("dev loop event %d: %v", devLoopCompleteEvent, entry)
			default:
				t.Logf("unknown event: %v", entry)
			}
		}
		return len(logEntries), buildCompleteEvent, devLoopCompleteEvent
	}
	logEntries, buildCompleteEvents, devLoopCompleteEvents := extractInfoFromFile(fName)

	// make sure we have exactly 1 build entry and 1 dev loop complete entry
	testutil.CheckDeepEqual(t, 2, logEntries)
	testutil.CheckDeepEqual(t, 1, buildCompleteEvents)
	testutil.CheckDeepEqual(t, 1, devLoopCompleteEvents)

	// Resaving should append the file.
	if err := SaveEventsToFile(fName); err != nil {
		t.Fatalf("error saving events to file: %v", err)
	}
	logEntries, buildCompleteEvents, devLoopCompleteEvents = extractInfoFromFile(fName)
	// Numbers double because appended.
	testutil.CheckDeepEqual(t, 4, logEntries)
	testutil.CheckDeepEqual(t, 2, buildCompleteEvents)
	testutil.CheckDeepEqual(t, 2, devLoopCompleteEvents)
}

func TestSaveLastLog(t *testing.T) {
	// Generate the file name to dump the file. File and directory should be created if it doesn't exist.
	fName := filepath.Join(os.TempDir(), "test", "logfile")
	t.Cleanup(func() { os.Remove(fName) })

	// add some events to the event log. Include irrelevant events to test that they are ignored
	handler.eventLog = []*proto.Event{
		{
			EventType: &proto.Event_BuildSubtaskEvent{},
		}, {
			EventType: &proto.Event_SkaffoldLogEvent{
				SkaffoldLogEvent: &proto.SkaffoldLogEvent{Message: "Message 1\n"},
			},
		}, {
			EventType: &proto.Event_DeploySubtaskEvent{},
		}, {
			EventType: &proto.Event_SkaffoldLogEvent{
				SkaffoldLogEvent: &proto.SkaffoldLogEvent{Message: "Message 2\n"},
			},
		}, {
			EventType: &proto.Event_PortEvent{},
		},
	}

	// save events to file
	if err := SaveLastLog(fName); err != nil {
		t.Fatalf("error saving log to file: %v", err)
	}

	// ensure that the events in the file match the event log
	b, err := os.ReadFile(fName)
	if err != nil {
		t.Fatalf("reading tmp file: %v", err)
	}

	// make sure that the contents of the file match the expected result.
	expectedText := `Message 1
Message 2
`
	testutil.CheckDeepEqual(t, expectedText, string(b))

	// save events to file again.
	if err := SaveLastLog(fName); err != nil {
		t.Fatalf("error saving log to file: %v", err)
	}

	// ensure that the events in the file match the event log, no append
	bAfter, err := os.ReadFile(fName)
	if err != nil {
		t.Fatalf("reading tmp file: %v", err)
	}

	testutil.CheckDeepEqual(t, expectedText, string(bAfter))
}

func TestLastLogFile(t *testing.T) {
	homeDir, _ := homedir.Dir()
	tests := []struct {
		name     string
		fp       string
		expected string
	}{
		{
			name:     "Empty string passed in",
			fp:       "",
			expected: filepath.Join(homeDir, ".skaffold", "last.log"),
		},
		{
			name:     "Non-empty string passed in",
			fp:       filepath.Join("/", "tmp"),
			expected: filepath.Join("/", "tmp"),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual, _ := lastLogFile(test.fp)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type config struct {
	pipes   []latest.Pipeline
	kubectx string
}

func (c config) GetKubeContext() string          { return c.kubectx }
func (c config) AutoBuild() bool                 { return true }
func (c config) AutoDeploy() bool                { return true }
func (c config) AutoSync() bool                  { return true }
func (c config) GetPipelines() []latest.Pipeline { return c.pipes }
func (c config) GetRunID() string                { return "run-id" }

func mockCfg(pipes []latest.Pipeline, kubectx string) config {
	return config{
		pipes:   pipes,
		kubectx: kubectx,
	}
}
