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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	protoV3 "github.com/GoogleContainerTools/skaffold/proto/v3"
	proto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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

	SubtaskIDNone = "-1"
)

const (
	BuildStartedEvent                 string = "BuildStartedEvent"
	BuildFailedEvent                  string = "BuildFailedEvent"
	BuildSucceededEvent               string = "BuildSucceededEvent"
	BuildCancelledEvent               string = "BuildCancelledEvent"
	DebuggingContainerStartedEvent    string = "DebuggingContainerStartedEvent"
	DebuggingContainerTerminatedEvent string = "DebuggingContainerTerminatedEvent"
	DeployStartedEvent                string = "DeployStartedEvent"
	DeployFailedEvent                 string = "DeployFailedEvent"
	DeploySucceededEvent              string = "DeploySucceededEvent"
	SkaffoldLogEvent                  string = "SkaffoldLogEvent"
	MetaEvent                         string = "MetaEvent"
	RenderStartedEvent                string = "RenderStartedEvent"
	RenderFailedEvent                 string = "RenderFailedEvent"
	RenderSucceededEvent              string = "RenderSucceededEvent"
	StatusCheckFailedEvent            string = "StatusCheckFailedEvent"
	StatusCheckStartedEvent           string = "StatusCheckStartedEvent"
	StatusCheckSucceededEvent         string = "StatusCheckSucceededEvent"
	TestFailedEvent                   string = "TestFailedEvent"
	TestSucceededEvent                string = "TestSucceededEvent"
	TestStartedEvent                  string = "TestStartedEvent"
	ApplicationLogEvent               string = "ApplicationLogEvent"
	PortForwardedEvent                string = "PortForwardedEvent"
	FileSyncEvent                     string = "FileSyncEvent"
	TaskStartedEvent                  string = "TaskStartedEvent"
	TaskFailedEvent                   string = "TaskFailedEvent"
	TaskCompletedEvent                string = "TaskCompletedEvent"
)

var handler = newHandler()

func newHandler() *eventHandler {
	h := &eventHandler{
		eventChan: make(chan *protoV3.Event),
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
	eventLog                []protoV3.Event
	logLock                 sync.Mutex
	applicationLogs         []protoV3.Event
	applicationLogsLock     sync.Mutex
	cfg                     Config
	iteration               int
	state                   protoV3.State
	stateLock               sync.Mutex
	eventChan               chan *protoV3.Event
	eventListeners          []*listener
	applicationLogListeners []*listener
}

type listener struct {
	callback func(*protoV3.Event) error
	errors   chan error
	closed   bool
}

func GetIteration() int {
	return handler.iteration
}

func GetState() (*protoV3.State, error) {
	state := handler.getState()
	return &state, nil
}

func ForEachEvent(callback func(*protoV3.Event) error) error {
	return handler.forEachEvent(callback)
}

func ForEachApplicationLog(callback func(*protoV3.Event) error) error {
	return handler.forEachApplicationLog(callback)
}

func Handle(event *protoV3.Event) error {
	if event != nil {
		handler.publishEventOnChannel(event)
	}
	return nil
}

func (ev *eventHandler) getState() protoV3.State {
	ev.stateLock.Lock()
	// Deep copy
	buf, _ := json.Marshal(ev.state)
	ev.stateLock.Unlock()

	var state protoV3.State
	json.Unmarshal(buf, &state)

	return state
}

func (ev *eventHandler) log(event *protoV3.Event, listeners *[]*listener, log *[]protoV3.Event, lock sync.Locker) {
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

func (ev *eventHandler) logEvent(event *protoV3.Event) {
	ev.log(event, &ev.eventListeners, &ev.eventLog, &ev.logLock)
}

func (ev *eventHandler) logApplicationLog(event *protoV3.Event) {
	ev.log(event, &ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock)
}

func (ev *eventHandler) forEach(listeners *[]*listener, log *[]protoV3.Event, lock sync.Locker, callback func(*protoV3.Event) error) error {
	listener := &listener{
		callback: callback,
		errors:   make(chan error),
	}

	lock.Lock()

	oldEvents := make([]protoV3.Event, len(*log))
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

func (ev *eventHandler) forEachEvent(callback func(*protoV3.Event) error) error {
	return ev.forEach(&ev.eventListeners, &ev.eventLog, &ev.logLock, callback)
}

func (ev *eventHandler) forEachApplicationLog(callback func(*protoV3.Event) error) error {
	return ev.forEach(&ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock, callback)
}

func emptyState(cfg Config) protoV3.State {
	builds := map[string]string{}
	for _, p := range cfg.GetPipelines() {
		for _, a := range p.Build.Artifacts {
			builds[a.ImageName] = NotStarted
		}
	}
	metadata := initializeMetadata(cfg.GetPipelines(), cfg.GetKubeContext(), cfg.GetRunID())
	return emptyStateWithArtifacts(builds, metadata, cfg.AutoBuild(), cfg.AutoDeploy(), cfg.AutoSync())
}

func emptyStateWithArtifacts(builds map[string]string, metadata *protoV3.Metadata, autoBuild, autoDeploy, autoSync bool) protoV3.State {
	return protoV3.State{
		BuildState: &protoV3.BuildState{
			Artifacts:   builds,
			AutoTrigger: autoBuild,
			StatusCode:  protoV3.StatusCode_OK,
		},
		TestState: &protoV3.TestState{
			Status:     NotStarted,
			StatusCode: protoV3.StatusCode_OK,
		},
		RenderState: &protoV3.RenderState{
			Status:     NotStarted,
			StatusCode: protoV3.StatusCode_OK,
		},
		DeployState: &protoV3.DeployState{
			Status:      NotStarted,
			AutoTrigger: autoDeploy,
			StatusCode:  protoV3.StatusCode_OK,
		},
		StatusCheckState: emptyStatusCheckState(),
		ForwardedPorts:   make(map[int32]*protoV3.PortForwardEvent),
		FileSyncState: &protoV3.FileSyncState{
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
	newState.DeployState.StatusCode = protoV3.StatusCode_OK
	newState.StatusCheckState = emptyStatusCheckState()
	newState.ForwardedPorts = map[int32]*protoV3.PortForwardEvent{}
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

func emptyStatusCheckState() *protoV3.StatusCheckState {
	return &protoV3.StatusCheckState{
		Status:     NotStarted,
		Resources:  map[string]string{},
		StatusCode: protoV3.StatusCode_OK,
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

		handler.applicationLogs = []protoV3.Event{}
	}

	event := &protoV3.TaskStartedEvent{
		Id:          fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:        string(task),
		Description: description,
		Iteration:   int32(handler.iteration),
		Status:      InProgress,
	}
	handler.handle(event.Id, event, TaskStartedEvent)
}

func TaskFailed(task constants.Phase, err error) {
	ae := sErrors.ActionableErrV3(handler.cfg, task, err)
	event := &protoV3.TaskFailedEvent{
		Id:            fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:          string(task),
		Iteration:     int32(handler.iteration),
		Status:        Failed,
		ActionableErr: ae,
	}
	handler.handle(event.Id, event, TaskFailedEvent)
}

func TaskSucceeded(task constants.Phase) {
	event := &protoV3.TaskCompletedEvent{
		Id:        fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:      string(task),
		Iteration: int32(handler.iteration),
		Status:    Succeeded,
	}
	handler.handle(event.Id, event, TaskCompletedEvent)
}

// PortForwarded notifies that a remote port has been forwarded locally.
func PortForwarded(localPort int32, remotePort util.IntOrString, podName, containerName, namespace string, portName string, resourceType, resourceName, address string) {
	event := &protoV3.PortForwardEvent{
		TaskId:        fmt.Sprintf("%s-%d", constants.PortForward, handler.iteration),
		LocalPort:     localPort,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		PortName:      portName,
		ResourceType:  resourceType,
		ResourceName:  resourceName,
		Address:       address,
		TargetPort: &protoV3.IntOrString{
			Type:   int32(remotePort.Type),
			IntVal: int32(remotePort.IntVal),
			StrVal: remotePort.StrVal,
		},
	}

	handler.handle(event.TaskId, event, PortForwardedEvent)
}

func (ev *eventHandler) setState(state protoV3.State) {
	ev.stateLock.Lock()
	ev.state = state
	ev.stateLock.Unlock()
}

func (ev *eventHandler) handle(id string, event proto.Message, eventtype string) {
	eventInAnyFormat := &anypb.Any{}
	anypb.MarshalFrom(eventInAnyFormat, event, proto.MarshalOptions{})

	containerEvent := &protoV3.Event{Id: uuid.New().String(), Type: eventtype, Data: eventInAnyFormat, Specversion: "1.0", Source: "skaffold.dev"}
	containerEvent.Datacontenttype = "application/protobuf"
	ev.publishEventOnChannel(containerEvent)
}

func (ev *eventHandler) publishEventOnChannel(event *protoV3.Event) {
	event.Time = timestamppb.Now()
	ev.eventChan <- event
	if event.Type == "terminationEvent" {
		// close the event channel indicating there are no more events to all the
		// receivers
		close(ev.eventChan)
	}
}

func (ev *eventHandler) handleExec(event *protoV3.Event) {
	switch event.Type {
	case ApplicationLogEvent:
		ev.logApplicationLog(event)
		return
	case BuildSucceededEvent:
		buildEvent := &protoV3.BuildSucceededEvent{}
		anypb.UnmarshalTo(event.Data, buildEvent, proto.UnmarshalOptions{})
		if buildEvent.Step == Build {
			ev.stateLock.Lock()
			ev.state.BuildState.Artifacts[buildEvent.Artifact] = buildEvent.Status
			ev.stateLock.Unlock()
		}
	case BuildStartedEvent:
		buildEvent := &protoV3.BuildStartedEvent{}
		anypb.UnmarshalTo(event.Data, buildEvent, proto.UnmarshalOptions{})
		fmt.Println(buildEvent)
		if buildEvent.Step == Build {
			ev.stateLock.Lock()
			ev.state.BuildState.Artifacts[buildEvent.Artifact] = buildEvent.Status
			ev.stateLock.Unlock()
		}
	case BuildFailedEvent:
		buildEvent := &protoV3.BuildFailedEvent{}
		anypb.UnmarshalTo(event.Data, buildEvent, proto.UnmarshalOptions{})
		if buildEvent.Step == Build {
			ev.stateLock.Lock()
			ev.state.BuildState.Artifacts[buildEvent.Artifact] = buildEvent.Status
			ev.stateLock.Unlock()
		}
	case BuildCancelledEvent:
		buildEvent := &protoV3.BuildCancelledEvent{}
		anypb.UnmarshalTo(event.Data, buildEvent, proto.UnmarshalOptions{})
		if buildEvent.Step == Build {
			ev.stateLock.Lock()
			ev.state.BuildState.Artifacts[buildEvent.Artifact] = buildEvent.Status
			ev.stateLock.Unlock()
		}
	case TestFailedEvent:
		te := &protoV3.TestFailedEvent{}
		anypb.UnmarshalTo(event.Data, te, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.TestState.Status = te.Status
		ev.stateLock.Unlock()
	case TestStartedEvent:
		te := &protoV3.TestStartedEvent{}
		anypb.UnmarshalTo(event.Data, te, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.TestState.Status = te.Status
		ev.stateLock.Unlock()
	case TestSucceededEvent:
		te := &protoV3.TestSucceededEvent{}
		anypb.UnmarshalTo(event.Data, te, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.TestState.Status = te.Status
		ev.stateLock.Unlock()
	case RenderFailedEvent:
		re := &protoV3.RenderFailedEvent{}
		anypb.UnmarshalTo(event.Data, re, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.RenderState.Status = re.Status
		ev.stateLock.Unlock()
	case RenderSucceededEvent:
		re := &protoV3.RenderSucceededEvent{}
		anypb.UnmarshalTo(event.Data, re, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.RenderState.Status = re.Status
		ev.stateLock.Unlock()
	case RenderStartedEvent:
		re := &protoV3.RenderStartedEvent{}
		anypb.UnmarshalTo(event.Data, re, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.RenderState.Status = re.Status
		ev.stateLock.Unlock()
	case DeployStartedEvent:
		de := &protoV3.DeployStartedEvent{}
		anypb.UnmarshalTo(event.Data, de, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
	case DeployFailedEvent:
		de := &protoV3.DeployFailedEvent{}
		anypb.UnmarshalTo(event.Data, de, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
	case DeploySucceededEvent:
		de := &protoV3.DeploySucceededEvent{}
		anypb.UnmarshalTo(event.Data, de, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
	case PortForwardedEvent:
		pe := &protoV3.PortForwardEvent{}
		anypb.UnmarshalTo(event.Data, pe, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		if ev.state.ForwardedPorts == nil {
			ev.state.ForwardedPorts = map[int32]*protoV3.PortForwardEvent{}
		}
		ev.state.ForwardedPorts[pe.LocalPort] = pe
		ev.stateLock.Unlock()
	case StatusCheckStartedEvent:
		se := &protoV3.StatusCheckStartedEvent{}
		anypb.UnmarshalTo(event.Data, se, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Resources[se.Resource] = se.Status
		ev.stateLock.Unlock()
	case StatusCheckSucceededEvent:
		se := &protoV3.StatusCheckSucceededEvent{}
		anypb.UnmarshalTo(event.Data, se, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Resources[se.Resource] = se.Status
		ev.stateLock.Unlock()
	case StatusCheckFailedEvent:
		se := &protoV3.StatusCheckFailedEvent{}
		anypb.UnmarshalTo(event.Data, se, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Resources[se.Resource] = se.Status
		ev.stateLock.Unlock()
	case FileSyncEvent:
		fse := &protoV3.FileSyncEvent{}
		anypb.UnmarshalTo(event.Data, fse, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.FileSyncState.Status = fse.Status
		ev.stateLock.Unlock()
	case DebuggingContainerStartedEvent:
		de := &protoV3.DebuggingContainerStartedEvent{}
		anypb.UnmarshalTo(event.Data, de, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		ev.state.DebuggingContainers = append(ev.state.DebuggingContainers, &protoV3.DebuggingContainerState{
			Id:            de.Id,
			TaskId:        de.TaskId,
			Status:        de.Status,
			PodName:       de.PodName,
			ContainerName: de.ContainerName,
			Namespace:     de.Namespace,
			Artifact:      de.Artifact,
			Runtime:       de.Runtime,
			WorkingDir:    de.WorkingDir,
			DebugPorts:    de.DebugPorts,
		})
		ev.stateLock.Unlock()
	case DebuggingContainerTerminatedEvent:
		de := &protoV3.DebuggingContainerTerminatedEvent{}
		anypb.UnmarshalTo(event.Data, de, proto.UnmarshalOptions{})
		ev.stateLock.Lock()
		n := 0
		for _, x := range ev.state.DebuggingContainers {
			if x.Namespace != de.Namespace || x.PodName != de.PodName || x.ContainerName != de.ContainerName {
				ev.state.DebuggingContainers[n] = x
				n++
			}
		}
		ev.state.DebuggingContainers = ev.state.DebuggingContainers[:n]
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
