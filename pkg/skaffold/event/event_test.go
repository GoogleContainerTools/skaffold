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

package event

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var targetPort = proto.IntOrString{Type: 0, IntVal: 2001}

func TestGetLogEvents(t *testing.T) {
	for step := 0; step < 1000; step++ {
		ev := newHandler()

		ev.logEvent(&proto.LogEntry{Entry: "OLD"})
		go func() {
			ev.logEvent(&proto.LogEntry{Entry: "FRESH"})
			ev.logEvent(&proto.LogEntry{Entry: "POISON PILL"})
		}()

		var received int32
		ev.forEachEvent(func(e *proto.LogEntry) error {
			if e.Entry == "POISON PILL" {
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

func TestDeployInProgress(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployInProgress()
	wait(t, func() bool { return handler.getState().DeployState.Status == InProgress })
}

func TestDeployFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployFailed(errors.New("BUG"))
	wait(t, func() bool {
		dState := handler.getState().DeployState
		return dState.Status == Failed && dState.StatusCode == proto.StatusCode_DEPLOY_UNKNOWN
	})
}

func TestDeployComplete(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
	DeployComplete()
	wait(t, func() bool {
		dState := handler.getState().DeployState
		return dState.Status == Complete && dState.StatusCode == proto.StatusCode_DEPLOY_SUCCESS
	})
}

func TestTestInProgress(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
	TestInProgress()
	wait(t, func() bool { return handler.getState().TestState.Status == InProgress })
}

func TestTestFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
	TestFailed("img", errors.New("BUG"))
	wait(t, func() bool {
		tState := handler.getState().TestState
		output.Yellow.Fprintf(os.Stdout, "Priya_1 tState is: %s): ", tState)
		return tState.Status == Failed && tState.StatusCode == proto.StatusCode_TEST_UNKNOWN
	})
}

func TestTestComplete(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
	TestComplete()
	wait(t, func() bool { return handler.getState().TestState.Status == Complete })
}

func TestBuildInProgress(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{Build: latest.BuildConfig{
		Artifacts: []*latest.Artifact{{
			ImageName: "img",
		}},
	}}}, "test"))

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildInProgress("img", "")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == InProgress })
}

func TestBuildFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{Build: latest.BuildConfig{
		Artifacts: []*latest.Artifact{{
			ImageName: "img",
		}},
	}}}, "test"))

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildFailed("img", "", errors.New("BUG"))
	wait(t, func() bool {
		bState := handler.getState().BuildState
		return bState.Artifacts["img"] == Failed
	})
}

func TestBuildComplete(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{Build: latest.BuildConfig{
		Artifacts: []*latest.Artifact{{
			ImageName: "img",
		}},
	}}}, "test"))

	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == NotStarted })
	BuildComplete("img", "")
	wait(t, func() bool { return handler.getState().BuildState.Artifacts["img"] == Complete })
}

func TestPortForwarded(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().ForwardedPorts[8080] == nil })
	PortForwarded(8080, schemautil.FromInt(8888), "pod", "container", "ns", "portname", "resourceType", "resourceName", "127.0.0.1")
	wait(t, func() bool {
		return handler.getState().ForwardedPorts[8080] != nil && handler.getState().ForwardedPorts[8080].RemotePort == 8888
	})

	wait(t, func() bool { return handler.getState().ForwardedPorts[8081] == nil })
	PortForwarded(8081, schemautil.FromString("http"), "pod", "container", "ns", "portname", "resourceType", "resourceName", "127.0.0.1")
	wait(t, func() bool { return handler.getState().ForwardedPorts[8081] != nil })
}

// Ensure that port-forward event handling deals with a nil State.ForwardedPorts map.
// See https://github.com/GoogleContainerTools/skaffold/issues/5612
func TestPortForwarded_handleNil(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))
	handler.setState(handler.getState())

	if handler.getState().ForwardedPorts != nil {
		t.Error("ForwardPorts should be a nil map")
	}
	PortForwarded(8080, schemautil.FromInt(8888), "pod", "container", "ns", "portname", "resourceType", "resourceName", "127.0.0.1")
	wait(t, func() bool { return handler.getState().ForwardedPorts[8080] != nil })
}

func TestResourceCheckEvent_handleNil(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))
	handler.setState(handler.getState())

	if handler.getState().StatusCheckState.Resources != nil {
		t.Error("Resources should be a nil map")
	}
	resourceStatusCheckEventSucceeded("pods")
	// Resources will be reset to map[string]string{} in handleExec
	wait(t, func() bool {
		_, ok := handler.getState().StatusCheckState.Resources["pods"]
		return ok
	})
}

func TestStatusCheckEventStarted(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	StatusCheckEventStarted()
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == Started })
}

func TestStatusCheckEventInProgress(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	StatusCheckEventInProgress("[2/5 deployment(s) are still pending]")
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == InProgress })
}

func TestStatusCheckEventSucceeded(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	statusCheckEventSucceeded()
	wait(t, func() bool { return handler.getState().StatusCheckState.Status == Succeeded })
}

func TestStatusCheckEventFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	StatusCheckEventEnded(proto.StatusCode_STATUSCHECK_FAILED_SCHEDULING, errors.New("one or more deployments failed"))
	wait(t, func() bool {
		state := handler.getState().StatusCheckState
		return state.Status == Failed && state.StatusCode == proto.StatusCode_STATUSCHECK_FAILED_SCHEDULING
	})
}

func TestResourceStatusCheckEventUpdated(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	ResourceStatusCheckEventUpdated("ns:pod/foo", &proto.ActionableErr{
		ErrCode: 509,
		Message: "image pull error",
	})
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == InProgress })
}

func TestResourceStatusCheckEventSucceeded(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventSucceeded("ns:pod/foo")
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Succeeded })
}

func TestResourceStatusCheckEventFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventFailed("ns:pod/foo", &proto.ActionableErr{
		ErrCode: 309,
		Message: "one or more deployments failed",
	})
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Failed })
}

func TestFileSyncInProgress(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncInProgress(5, "image")
	wait(t, func() bool { return handler.getState().FileSyncState.Status == InProgress })
}

func TestFileSyncFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncFailed(5, "image", errors.New("BUG"))
	wait(t, func() bool { return handler.getState().FileSyncState.Status == Failed })
}

func TestFileSyncSucceeded(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().FileSyncState.Status == NotStarted })
	FileSyncSucceeded(5, "image")
	wait(t, func() bool { return handler.getState().FileSyncState.Status == Succeeded })
}

func TestDebuggingContainer(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	found := func() bool {
		for _, dc := range handler.getState().DebuggingContainers {
			if dc.Namespace == "ns" && dc.PodName == "pod" && dc.ContainerName == "container" {
				return true
			}
		}
		return false
	}
	notFound := func() bool { return !found() }
	wait(t, notFound)
	DebuggingContainerStarted("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, found)
	DebuggingContainerTerminated("pod", "container", "ns", "artifact", "runtime", "/", nil)
	wait(t, notFound)
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
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortEvent{
			2001: {
				LocalPort:  2000,
				RemotePort: 2001,
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
		DeployState:      &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted, Resources: map[string]string{}},
		FileSyncState:    &proto.FileSyncState{Status: NotStarted},
	}
	testutil.CheckDeepEqual(t, &expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestResetStateOnDeploy(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortEvent{
			2001: {
				LocalPort:  2000,
				RemotePort: 2001,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
	}
	ResetStateOnDeploy()
	expected := &proto.State{
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
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestEmptyStateCheckState(t *testing.T) {
	actual := emptyStatusCheckState()
	expected := &proto.StatusCheckState{Status: NotStarted,
		Resources: map[string]string{},
	}
	testutil.CheckDeepEqual(t, expected, actual, cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestUpdateStateAutoTriggers(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: false,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: false},
		ForwardedPorts: map[int32]*proto.PortEvent{
			2001: {
				LocalPort:  2000,
				RemotePort: 2001,
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
		ForwardedPorts: map[int32]*proto.PortEvent{
			2001: {
				LocalPort:  2000,
				RemotePort: 2001,
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
	testutil.CheckDeepEqual(t, &expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestDevLoopFailedInPhase(t *testing.T) {
	tcs := []struct {
		description string
		state       *proto.State
		phase       constants.Phase
		waitFn      func() bool
	}{
		{
			description: "build failed",
			state: &proto.State{
				BuildState: &proto.BuildState{StatusCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED},
			},
			phase: constants.Build,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				return logEntry.Entry == fmt.Sprintf("Update failed with error code %v", proto.StatusCode_BUILD_PUSH_ACCESS_DENIED)
			},
		},
		{
			description: "test failed",
			state: &proto.State{
				BuildState: &proto.BuildState{},
				TestState:  &proto.TestState{StatusCode: proto.StatusCode_TEST_UNKNOWN},
			},
			phase: constants.Test,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				return logEntry.Entry == fmt.Sprintf("Update failed with error code %v", proto.StatusCode_TEST_UNKNOWN)
			},
		},
		{
			description: "deploy failed",
			state: &proto.State{
				BuildState:  &proto.BuildState{},
				DeployState: &proto.DeployState{StatusCode: proto.StatusCode_DEPLOY_UNKNOWN},
			},
			phase: constants.Deploy,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				return logEntry.Entry == fmt.Sprintf("Update failed with error code %v", proto.StatusCode_DEPLOY_UNKNOWN)
			},
		},
		{
			description: "status check failed",
			state: &proto.State{
				BuildState:       &proto.BuildState{},
				TestState:        &proto.TestState{StatusCode: proto.StatusCode_TEST_SUCCESS},
				DeployState:      &proto.DeployState{StatusCode: proto.StatusCode_DEPLOY_SUCCESS},
				StatusCheckState: &proto.StatusCheckState{StatusCode: proto.StatusCode_STATUSCHECK_UNHEALTHY},
			},
			phase: constants.Deploy,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				return logEntry.Entry == fmt.Sprintf("Update failed with error code %v", proto.StatusCode_STATUSCHECK_UNHEALTHY)
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			handler.setState(tc.state)
			DevLoopFailedInPhase(0, tc.phase, errors.New("random error"))
			wait(t, tc.waitFn)
		})
	}
}

func TestSaveEventsToFile(t *testing.T) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("getting temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	if err := f.Close(); err != nil {
		t.Fatalf("error closing tmp file: %v", err)
	}

	// add some events to the event log
	handler.eventLog = []*proto.LogEntry{
		{
			Event: &proto.Event{EventType: &proto.Event_BuildEvent{}},
		}, {
			Event: &proto.Event{EventType: &proto.Event_DevLoopEvent{}},
		},
	}

	// save events to file
	if err := SaveEventsToFile(f.Name()); err != nil {
		t.Fatalf("error saving events to file: %v", err)
	}

	// ensure that the events in the file match the event log
	contents, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("reading tmp file: %v", err)
	}

	var logEntries []*proto.LogEntry
	entries := strings.Split(string(contents), "\n")
	for _, e := range entries {
		if e == "" {
			continue
		}
		var logEntry proto.LogEntry
		if err := jsonpb.UnmarshalString(e, &logEntry); err != nil {
			t.Errorf("error converting http response %s to proto: %s", e, err.Error())
		}
		logEntries = append(logEntries, &logEntry)
	}

	buildCompleteEvent, devLoopCompleteEvent := 0, 0
	for _, entry := range logEntries {
		t.Log(entry.Event.GetEventType())
		switch entry.Event.GetEventType().(type) {
		case *proto.Event_BuildEvent:
			buildCompleteEvent++
			t.Logf("build event %d: %v", buildCompleteEvent, entry.Event)
		case *proto.Event_DevLoopEvent:
			devLoopCompleteEvent++
			t.Logf("dev loop event %d: %v", devLoopCompleteEvent, entry.Event)
		default:
			t.Logf("unknown event: %v", entry.Event)
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
