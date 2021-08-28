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

package v3

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
	proto1 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var targetPort = proto.IntOrString{Type: 0, IntVal: 2001}

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 1000; step++ {
		eventInAnyFormat := &anypb.Any{}

		anypb.MarshalFrom(eventInAnyFormat, &proto.SkaffoldLogEvent{
			Message: "OLD"}, proto1.MarshalOptions{})

		ev := newHandler()
		ev.logEvent(&proto.Event{
			Type: SkaffoldLogEvent, Data: eventInAnyFormat})

		go func() {
			localEvent1 := &anypb.Any{}
			anypb.MarshalFrom(localEvent1, &proto.SkaffoldLogEvent{
				Message: "FRESH"}, proto1.MarshalOptions{})
			ev.logEvent(&proto.Event{
				Type: SkaffoldLogEvent, Data: localEvent1})

			localEvent2 := &anypb.Any{}
			anypb.MarshalFrom(localEvent2, &proto.SkaffoldLogEvent{
				Message: "POISON PILL"}, proto1.MarshalOptions{})
			ev.logEvent(&proto.Event{
				Type: SkaffoldLogEvent, Data: localEvent2})

		}()

		var received int32
		ev.forEachEvent(func(e *proto.Event) error {
			se := &proto.SkaffoldLogEvent{}
			anypb.UnmarshalTo(e.Data, se, proto1.UnmarshalOptions{})

			if se.Message == "POISON PILL" {
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
	ev.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

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

func TestResetStateOnBuild(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		RenderState: &proto.RenderState{Status: Complete},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState:    &proto.FileSyncState{Status: Succeeded},
	}

	ResetStateOnBuild()
	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": NotStarted,
			},
		},
		TestState:        &proto.TestState{Status: NotStarted},
		RenderState:      &proto.RenderState{Status: NotStarted},
		DeployState:      &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted, Resources: map[string]string{}},
		FileSyncState:    &proto.FileSyncState{Status: NotStarted},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}

func TestResetStateOnDeploy(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
	}
	ResetStateOnDeploy()
	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted,
			Resources: map[string]string{},
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}

func TestEmptyStateCheckState(t *testing.T) {
	actual := emptyStatusCheckState()
	expected := &proto.StatusCheckState{Status: NotStarted,
		Resources: map[string]string{},
	}
	testutil.CheckDeepEqual(t, expected, actual, cmpopts.EquateEmpty())
}

func TestUpdateStateAutoTriggers(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: false,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: false},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState: &proto.FileSyncState{
			Status:      "Complete",
			AutoTrigger: false,
		},
	}
	UpdateStateAutoBuildTrigger(true)
	UpdateStateAutoDeployTrigger(true)
	UpdateStateAutoSyncTrigger(true)

	expected := proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: true,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: true},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState: &proto.FileSyncState{
			Status:      "Complete",
			AutoTrigger: true,
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty())
}

func TestTaskFailed(t *testing.T) {
	tcs := []struct {
		description string
		state       proto.State
		phase       constants.Phase
		waitFn      func() bool
	}{
		{
			description: "build failed",
			phase:       constants.Build,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()

				if logEntry.Type == TaskFailedEvent {
					taskFailedEvent := &proto.TaskFailedEvent{}
					anypb.UnmarshalTo(logEntry.Data, taskFailedEvent, proto1.UnmarshalOptions{})
					return taskFailedEvent != nil && taskFailedEvent.Id == "Build-0"
				}
				return false
			},
		},
		{
			description: "deploy failed",
			phase:       constants.Deploy,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				if logEntry.Type == TaskFailedEvent {
					taskFailedEvent := &proto.TaskFailedEvent{}
					anypb.UnmarshalTo(logEntry.Data, taskFailedEvent, proto1.UnmarshalOptions{})
					return taskFailedEvent != nil && taskFailedEvent.Id == "Deploy-0"
				}
				return false
			},
		},
		{
			description: "status check failed",
			phase:       constants.StatusCheck,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				if logEntry.Type == TaskFailedEvent {
					taskFailedEvent := &proto.TaskFailedEvent{}
					anypb.UnmarshalTo(logEntry.Data, taskFailedEvent, proto1.UnmarshalOptions{})
					return taskFailedEvent != nil && taskFailedEvent.Id == "StatusCheck-0"
				}
				return false
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			TaskFailed(tc.phase, errors.New("random error"))
			wait(t, tc.waitFn)
		})
	}
}

func TestAutoTriggerDiff(t *testing.T) {
	tests := []struct {
		description  string
		phase        constants.Phase
		handlerState proto.State
		val          bool
		expected     bool
	}{
		{
			description: "build needs update",
			phase:       constants.Build,
			val:         true,
			handlerState: proto.State{
				BuildState: &proto.BuildState{
					AutoTrigger: false,
				},
			},
			expected: true,
		},
		{
			description: "deploy doesn't need update",
			phase:       constants.Deploy,
			val:         true,
			handlerState: proto.State{
				BuildState: &proto.BuildState{
					AutoTrigger: false,
				},
				DeployState: &proto.DeployState{
					AutoTrigger: true,
				},
			},
			expected: false,
		},
		{
			description: "sync needs update",
			phase:       constants.Sync,
			val:         false,
			handlerState: proto.State{
				FileSyncState: &proto.FileSyncState{
					AutoTrigger: true,
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Setup handler state
			handler.setState(test.handlerState)

			got, err := AutoTriggerDiff(test.phase, test.val)
			if err != nil {
				t.Fail()
			}

			t.CheckDeepEqual(test.expected, got)
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

	buildEvent := &anypb.Any{}
	anypb.MarshalFrom(buildEvent, &proto.BuildSucceededEvent{}, proto1.MarshalOptions{})

	taskEvent := &anypb.Any{}
	anypb.MarshalFrom(taskEvent, &proto.TaskCompletedEvent{}, proto1.MarshalOptions{})

	// add some events to the event log
	handler.eventLog = []proto.Event{
		{
			Data: buildEvent,
			Type: BuildSucceededEvent,
		}, {
			Data: taskEvent,
			Type: TaskCompletedEvent,
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
		////t.Log(entry.GetEventType())

		switch entry.Type {
		case BuildSucceededEvent:
			buildCompleteEvent++
			t.Logf("build event %d: %v", buildCompleteEvent, entry)
		case TaskCompletedEvent:
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
	pipes   []latestV1.Pipeline
	kubectx string
}

func (c config) GetKubeContext() string            { return c.kubectx }
func (c config) AutoBuild() bool                   { return true }
func (c config) AutoDeploy() bool                  { return true }
func (c config) AutoSync() bool                    { return true }
func (c config) GetPipelines() []latestV1.Pipeline { return c.pipes }
func (c config) GetRunID() string                  { return "run-id" }

func mockCfg(pipes []latestV1.Pipeline, kubectx string) config {
	return config{
		pipes:   pipes,
		kubectx: kubectx,
	}
}
