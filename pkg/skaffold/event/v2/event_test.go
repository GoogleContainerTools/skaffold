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

package v2

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/go-cmp/cmp/cmpopts"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

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

func TestGetState(t *testing.T) {
	ev := newHandler()
	ev.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	ev.stateLock.Lock()
	ev.state.BuildState.Artifacts["img"] = Complete
	ev.stateLock.Unlock()

	state := ev.getState()

	testutil.CheckDeepEqual(t, Complete, state.BuildState.Artifacts["img"])
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

func TestEmptyStateCheckState(t *testing.T) {
	actual := emptyStatusCheckState()
	expected := &proto.StatusCheckState{Status: NotStarted,
		Resources: map[string]string{},
	}
	testutil.CheckDeepEqual(t, expected, actual, cmpopts.EquateEmpty())
}

func TestTaskFailed(t *testing.T) {
	tcs := []struct {
		description string
		state       proto.State
		phase       sErrors.Phase
		iteration   int
		waitFn      func() bool
	}{
		{
			description: "build failed",
			phase:       sErrors.Build,
			iteration:   0,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "Build-0"
			},
		},
		{
			description: "deploy failed",
			phase:       sErrors.Deploy,
			iteration:   1,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "Deploy-1"
			},
		},
		{
			description: "status check failed",
			phase:       sErrors.StatusCheck,
			iteration:   2,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "StatusCheck-2"
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			TaskFailed(tc.phase, tc.iteration, errors.New("random error"))
			wait(t, tc.waitFn)
		})
	}
}

func TestSaveEventsToFile(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("getting temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	if err := f.Close(); err != nil {
		t.Fatalf("error closing tmp file: %v", err)
	}

	// add some events to the event log
	handler.eventLog = []proto.Event{
		{
			EventType: &proto.Event_BuildSubtaskEvent{},
		}, {
			EventType: &proto.Event_TaskEvent{},
		},
	}

	// save events to file
	if err := SaveEventsToFile(f.Name()); err != nil {
		t.Fatalf("error saving events to file: %v", err)
	}

	// ensure that the events in the file match the event log
	contents, err := ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("reading tmp file: %v", err)
	}

	var logEntries []proto.Event
	entries := strings.Split(string(contents), "\n")
	for _, e := range entries {
		if e == "" {
			continue
		}
		var logEntry proto.Event
		if err := jsonpb.UnmarshalString(e, &logEntry); err != nil {
			t.Errorf("error converting http response %s to proto: %s", e, err.Error())
		}
		logEntries = append(logEntries, logEntry)
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

	// make sure we have exactly 1 build entry and 1 dev loop complete entry
	testutil.CheckDeepEqual(t, 2, len(logEntries))
	testutil.CheckDeepEqual(t, 1, buildCompleteEvent)
	testutil.CheckDeepEqual(t, 1, devLoopCompleteEvent)
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

func mockCfg(pipes []latest.Pipeline, kubectx string) config {
	return config{
		pipes:   pipes,
		kubectx: kubectx,
	}
}
