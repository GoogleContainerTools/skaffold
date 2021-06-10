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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

const (
	NotStarted = "NotStarted"
	InProgress = "InProgress"
	Complete   = "Complete"
	Failed     = "Failed"
	Info       = "Information"
	Started    = "Started"
	Succeeded  = "Succeeded"
	Terminated = "Terminated"
	Canceled   = "Canceled"
)

var handler = newHandler()

func newHandler() *eventHandler {
	h := &eventHandler{
		eventChan: make(chan *proto.Event),
	}
	go func() {
		for {
			ev, open := <-h.eventChan
			if !open {
				break
			}
			h.handleExec(ev)
		}
	}()
	return h
}

type eventHandler struct {
	eventLog            []proto.Event
	logLock             sync.Mutex
	applicationLogs     []proto.Event
	applicationLogsLock sync.Mutex
	skaffoldLogs        []proto.Event
	skaffoldLogsLock    sync.Mutex
	cfg                 Config

	iteration               int
	state                   proto.State
	stateLock               sync.Mutex
	eventChan               chan *proto.Event
	eventListeners          []*listener
	applicationLogListeners []*listener
	skaffoldLogListeners    []*listener
}

type listener struct {
	callback func(*proto.Event) error
	errors   chan error
	closed   bool
}

func GetIteration() int {
	return handler.iteration
}

func GetState() (*proto.State, error) {
	state := handler.getState()
	return &state, nil
}

func ForEachEvent(callback func(*proto.Event) error) error {
	return handler.forEachEvent(callback)
}

func ForEachApplicationLog(callback func(*proto.Event) error) error {
	return handler.forEachApplicationLog(callback)
}

func ForEachSkaffoldLog(callback func(*proto.Event) error) error {
	return handler.forEachSkaffoldLog(callback)
}

func Handle(event *proto.Event) error {
	if event != nil {
		handler.handle(event)
	}
	return nil
}

func (ev *eventHandler) getState() proto.State {
	ev.stateLock.Lock()
	// Deep copy
	buf, _ := json.Marshal(ev.state)
	ev.stateLock.Unlock()

	var state proto.State
	json.Unmarshal(buf, &state)

	return state
}

func (ev *eventHandler) log(event *proto.Event, listeners *[]*listener, log *[]proto.Event, lock sync.Locker) {
	lock.Lock()

	for _, listener := range *listeners {
		if listener.closed {
			continue
		}

		if err := listener.callback(event); err != nil {
			listener.errors <- err
			listener.closed = true
		}
	}
	*log = append(*log, *event)

	lock.Unlock()
}

func (ev *eventHandler) logEvent(event *proto.Event) {
	ev.log(event, &ev.eventListeners, &ev.eventLog, &ev.logLock)
}

func (ev *eventHandler) logApplicationLog(event *proto.Event) {
	ev.log(event, &ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock)
}

func (ev *eventHandler) logSkaffoldLog(event *proto.Event) {
	ev.log(event, &ev.skaffoldLogListeners, &ev.skaffoldLogs, &ev.skaffoldLogsLock)
}

func (ev *eventHandler) forEach(listeners *[]*listener, log *[]proto.Event, lock sync.Locker, callback func(*proto.Event) error) error {
	listener := &listener{
		callback: callback,
		errors:   make(chan error),
	}

	lock.Lock()

	oldEvents := make([]proto.Event, len(*log))
	copy(oldEvents, *log)
	*listeners = append(*listeners, listener)

	lock.Unlock()

	for i := range oldEvents {
		if err := callback(&oldEvents[i]); err != nil {
			// listener should maybe be closed
			return err
		}
	}

	return <-listener.errors
}

func (ev *eventHandler) forEachEvent(callback func(*proto.Event) error) error {
	return ev.forEach(&ev.eventListeners, &ev.eventLog, &ev.logLock, callback)
}

func (ev *eventHandler) forEachApplicationLog(callback func(*proto.Event) error) error {
	return ev.forEach(&ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock, callback)
}

func (ev *eventHandler) forEachSkaffoldLog(callback func(*proto.Event) error) error {
	return ev.forEach(&ev.skaffoldLogListeners, &ev.skaffoldLogs, &ev.skaffoldLogsLock, callback)
}

func emptyState(cfg Config) proto.State {
	builds := map[string]string{}
	for _, p := range cfg.GetPipelines() {
		for _, a := range p.Build.Artifacts {
			builds[a.ImageName] = NotStarted
		}
	}
	metadata := initializeMetadata(cfg.GetPipelines(), cfg.GetKubeContext())
	return emptyStateWithArtifacts(builds, metadata, cfg.AutoBuild(), cfg.AutoDeploy(), cfg.AutoSync())
}

func emptyStateWithArtifacts(builds map[string]string, metadata *proto.Metadata, autoBuild, autoDeploy, autoSync bool) proto.State {
	return proto.State{
		BuildState: &proto.BuildState{
			Artifacts:   builds,
			AutoTrigger: autoBuild,
			StatusCode:  proto.StatusCode_OK,
		},
		TestState: &proto.TestState{
			Status:     NotStarted,
			StatusCode: proto.StatusCode_OK,
		},
		DeployState: &proto.DeployState{
			Status:      NotStarted,
			AutoTrigger: autoDeploy,
			StatusCode:  proto.StatusCode_OK,
		},
		StatusCheckState: emptyStatusCheckState(),
		ForwardedPorts:   make(map[int32]*proto.PortForwardEvent),
		FileSyncState: &proto.FileSyncState{
			Status:      NotStarted,
			AutoTrigger: autoSync,
		},
		Metadata: metadata,
	}
}

// ResetStateOnBuild resets the build, test, deploy and sync state
func ResetStateOnBuild() {
	builds := map[string]string{}
	for k := range handler.getState().BuildState.Artifacts {
		builds[k] = NotStarted
	}
	autoBuild, autoDeploy, autoSync := handler.getState().BuildState.AutoTrigger, handler.getState().DeployState.AutoTrigger, handler.getState().FileSyncState.AutoTrigger
	newState := emptyStateWithArtifacts(builds, handler.getState().Metadata, autoBuild, autoDeploy, autoSync)
	handler.setState(newState)
}

// ResetStateOnTest resets the test, deploy, sync and status check state
func ResetStateOnTest() {
	newState := handler.getState()
	newState.TestState.Status = NotStarted
	handler.setState(newState)
}

// ResetStateOnDeploy resets the deploy, sync and status check state
func ResetStateOnDeploy() {
	newState := handler.getState()
	newState.DeployState.Status = NotStarted
	newState.DeployState.StatusCode = proto.StatusCode_OK
	newState.StatusCheckState = emptyStatusCheckState()
	newState.ForwardedPorts = map[int32]*proto.PortForwardEvent{}
	newState.DebuggingContainers = nil
	handler.setState(newState)
}

func UpdateStateAutoBuildTrigger(t bool) {
	newState := handler.getState()
	newState.BuildState.AutoTrigger = t
	handler.setState(newState)
}

func UpdateStateAutoDeployTrigger(t bool) {
	newState := handler.getState()
	newState.DeployState.AutoTrigger = t
	handler.setState(newState)
}

func UpdateStateAutoSyncTrigger(t bool) {
	newState := handler.getState()
	newState.FileSyncState.AutoTrigger = t
	handler.setState(newState)
}

func emptyStatusCheckState() *proto.StatusCheckState {
	return &proto.StatusCheckState{
		Status:     NotStarted,
		Resources:  map[string]string{},
		StatusCode: proto.StatusCode_OK,
	}
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
func InitializeState(cfg Config) {
	handler.cfg = cfg
	handler.setState(emptyState(cfg))
}

func AutoTriggerDiff(phase constants.Phase, val bool) (bool, error) {
	switch phase {
	case constants.Build:
		return val != handler.getState().BuildState.AutoTrigger, nil
	case constants.Sync:
		return val != handler.getState().FileSyncState.AutoTrigger, nil
	case constants.Deploy:
		return val != handler.getState().DeployState.AutoTrigger, nil
	default:
		return false, fmt.Errorf("unknown Phase %v not found in handler state", phase)
	}
}

func TaskInProgress(task constants.Phase, description string) {
	// Special casing to increment iteration and clear application and skaffold logs
	if task == constants.DevLoop {
		handler.iteration++

		handler.applicationLogs = []proto.Event{}
		handler.skaffoldLogs = []proto.Event{}
	}

	handler.handleTaskEvent(&proto.TaskEvent{
		Id:          fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:        string(task),
		Description: description,
		Iteration:   int32(handler.iteration),
		Status:      InProgress,
	})
}

func TaskFailed(task constants.Phase, err error) {
	ae := sErrors.ActionableErrV2(handler.cfg, task, err)
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:            fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:          string(task),
		Iteration:     int32(handler.iteration),
		Status:        Failed,
		ActionableErr: ae,
	})
}

func TaskSucceeded(task constants.Phase) {
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:        fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:      string(task),
		Iteration: int32(handler.iteration),
		Status:    Succeeded,
	})
}

// PortForwarded notifies that a remote port has been forwarded locally.
func PortForwarded(localPort int32, remotePort util.IntOrString, podName, containerName, namespace string, portName string, resourceType, resourceName, address string) {
	event := proto.PortForwardEvent{
		TaskId:        fmt.Sprintf("%s-%d", constants.PortForward, handler.iteration),
		LocalPort:     localPort,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		PortName:      portName,
		ResourceType:  resourceType,
		ResourceName:  resourceName,
		Address:       address,
		TargetPort: &proto.IntOrString{
			Type:   int32(remotePort.Type),
			IntVal: int32(remotePort.IntVal),
			StrVal: remotePort.StrVal,
		},
	}
	handler.handle(&proto.Event{
		EventType: &proto.Event_PortEvent{
			PortEvent: &event,
		},
	})
}

func (ev *eventHandler) setState(state proto.State) {
	ev.stateLock.Lock()
	ev.state = state
	ev.stateLock.Unlock()
}

func (ev *eventHandler) handle(event *proto.Event) {
	go func(t *timestamp.Timestamp) {
		event.Timestamp = t
		ev.eventChan <- event
		if _, ok := event.GetEventType().(*proto.Event_TerminationEvent); ok {
			// close the event channel indicating there are no more events to all the
			// receivers
			close(ev.eventChan)
		}
	}(ptypes.TimestampNow())
}

func (ev *eventHandler) handleTaskEvent(e *proto.TaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_TaskEvent{
			TaskEvent: e,
		},
	})
}

func (ev *eventHandler) handleExec(event *proto.Event) {
	switch e := event.GetEventType().(type) {
	case *proto.Event_ApplicationLogEvent:
		ev.logApplicationLog(event)
		return
	case *proto.Event_SkaffoldLogEvent:
		ev.logSkaffoldLog(event)
		return
	case *proto.Event_BuildSubtaskEvent:
		be := e.BuildSubtaskEvent
		if be.Step == Build {
			ev.stateLock.Lock()
			ev.state.BuildState.Artifacts[be.Artifact] = be.Status
			ev.stateLock.Unlock()
		}
	case *proto.Event_TestEvent:
		te := e.TestEvent
		ev.stateLock.Lock()
		ev.state.TestState.Status = te.Status
		ev.stateLock.Unlock()
	case *proto.Event_DeploySubtaskEvent:
		de := e.DeploySubtaskEvent
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
	case *proto.Event_PortEvent:
		pe := e.PortEvent
		ev.stateLock.Lock()
		if ev.state.ForwardedPorts == nil {
			ev.state.ForwardedPorts = map[int32]*proto.PortForwardEvent{}
		}
		ev.state.ForwardedPorts[pe.LocalPort] = pe
		ev.stateLock.Unlock()
	case *proto.Event_StatusCheckSubtaskEvent:
		se := e.StatusCheckSubtaskEvent
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Resources[se.Resource] = se.Status
		ev.stateLock.Unlock()
	case *proto.Event_FileSyncEvent:
		fse := e.FileSyncEvent
		ev.stateLock.Lock()
		ev.state.FileSyncState.Status = fse.Status
		ev.stateLock.Unlock()
	case *proto.Event_DebuggingContainerEvent:
		de := e.DebuggingContainerEvent
		ev.stateLock.Lock()
		switch de.Status {
		case Started:
			ev.state.DebuggingContainers = append(ev.state.DebuggingContainers, de)
		case Terminated:
			n := 0
			for _, x := range ev.state.DebuggingContainers {
				if x.Namespace != de.Namespace || x.PodName != de.PodName || x.ContainerName != de.ContainerName {
					ev.state.DebuggingContainers[n] = x
					n++
				}
			}
			ev.state.DebuggingContainers = ev.state.DebuggingContainers[:n]
		}
		ev.stateLock.Unlock()
	}
	ev.logEvent(event)
}

// SaveEventsToFile saves the current event log to the filepath provided
func SaveEventsToFile(fp string) error {
	handler.logLock.Lock()
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", fp, err)
	}
	defer f.Close()
	marshaller := jsonpb.Marshaler{}
	for _, ev := range handler.eventLog {
		contents := bytes.NewBuffer([]byte{})
		if err := marshaller.Marshal(contents, &ev); err != nil {
			return fmt.Errorf("marshalling event: %w", err)
		}
		if _, err := f.WriteString(contents.String() + "\n"); err != nil {
			return fmt.Errorf("writing string: %w", err)
		}
	}
	handler.logLock.Unlock()
	return nil
}
