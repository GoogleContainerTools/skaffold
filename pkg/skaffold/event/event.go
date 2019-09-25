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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/ptypes"
)

const (
	NotStarted = "Not Started"
	InProgress = "In Progress"
	Complete   = "Complete"
	Failed     = "Failed"
	Info       = "Information"
	Started    = "Started"
	Succeeded  = "Succeeded"
)

var handler = &eventHandler{}

type eventHandler struct {
	eventLog []proto.LogEntry
	logLock  sync.Mutex

	state     proto.State
	stateLock sync.Mutex

	listeners []*listener
}

type listener struct {
	callback func(*proto.LogEntry) error
	errors   chan error
	closed   bool
}

func GetState() (*proto.State, error) {
	state := handler.getState()
	return &state, nil
}

func ForEachEvent(callback func(*proto.LogEntry) error) error {
	return handler.forEachEvent(callback)
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

func (ev *eventHandler) logEvent(entry proto.LogEntry) {
	ev.logLock.Lock()

	for _, listener := range ev.listeners {
		if listener.closed {
			continue
		}

		if err := listener.callback(&entry); err != nil {
			listener.errors <- err
			listener.closed = true
		}
	}
	ev.eventLog = append(ev.eventLog, entry)

	ev.logLock.Unlock()
}

func (ev *eventHandler) forEachEvent(callback func(*proto.LogEntry) error) error {
	listener := &listener{
		callback: callback,
		errors:   make(chan error),
	}

	ev.logLock.Lock()

	oldEvents := make([]proto.LogEntry, len(ev.eventLog))
	copy(oldEvents, ev.eventLog)
	ev.listeners = append(ev.listeners, listener)

	ev.logLock.Unlock()

	for i := range oldEvents {
		if err := callback(&oldEvents[i]); err != nil {
			// listener should maybe be closed
			return err
		}
	}

	return <-listener.errors
}

func emptyState(build latest.BuildConfig) proto.State {
	builds := map[string]string{}
	for _, a := range build.Artifacts {
		builds[a.ImageName] = NotStarted
	}
	return emptyStateWithArtifacts(builds)
}

func emptyStateWithArtifacts(builds map[string]string) proto.State {
	return proto.State{
		BuildState: &proto.BuildState{
			Artifacts: builds,
		},
		DeployState: &proto.DeployState{
			Status: NotStarted,
		},
		StatusCheckState: &proto.StatusCheckState{
			Status:    NotStarted,
			Resources: map[string]string{},
		},
		ForwardedPorts: make(map[int32]*proto.PortEvent),
	}
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
func InitializeState(build latest.BuildConfig) {
	handler.setState(emptyState(build))
}

// DeployInProgress notifies that a deployment has been started.
func DeployInProgress() {
	handler.handleDeployEvent(&proto.DeployEvent{Status: InProgress})
}

// DeployFailed notifies that non-fatal errors were encountered during a deployment.
func DeployFailed(err error) {
	handler.handleDeployEvent(&proto.DeployEvent{Status: Failed, Err: err.Error()})
}

// DeployEvent notifies that a deployment of non fatal interesting errors during deploy.
func DeployInfoEvent(err error) {
	handler.handleDeployEvent(&proto.DeployEvent{Status: Info, Err: err.Error()})
}

func StatusCheckEventSucceeded() {
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status: Succeeded,
	})
}

func StatusCheckEventFailed(err error) {
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status: Failed,
		Err:    err.Error(),
	})
}

func StatusCheckEventStarted() {
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status: Started,
	})
}

func StatusCheckEventInProgress(s string) {
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status:  InProgress,
		Message: s,
	})
}

func ResourceStatusCheckEventSucceeded(r string) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource: r,
		Status:   Succeeded,
		Message:  Succeeded,
	})
}

func ResourceStatusCheckEventFailed(r string, err error) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource: r,
		Status:   Failed,
		Err:      err.Error(),
	})
}

func ResourceStatusCheckEventUpdated(r string, status string) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource: r,
		Status:   InProgress,
		Message:  status,
	})
}

// DeployComplete notifies that a deployment has completed.
func DeployComplete() {
	handler.handleDeployEvent(&proto.DeployEvent{Status: Complete})
}

// BuildInProgress notifies that a build has been started.
func BuildInProgress(imageName string) {
	handler.handleBuildEvent(&proto.BuildEvent{Artifact: imageName, Status: InProgress})
}

// BuildFailed notifies that a build has failed.
func BuildFailed(imageName string, err error) {
	handler.handleBuildEvent(&proto.BuildEvent{Artifact: imageName, Status: Failed, Err: err.Error()})
}

// BuildComplete notifies that a build has completed.
func BuildComplete(imageName string) {
	handler.handleBuildEvent(&proto.BuildEvent{Artifact: imageName, Status: Complete})
}

// PortForwarded notifies that a remote port has been forwarded locally.
func PortForwarded(localPort, remotePort int32, podName, containerName, namespace string, portName string, resourceType, resourceName string) {
	go handler.handle(&proto.Event{
		EventType: &proto.Event_PortEvent{
			PortEvent: &proto.PortEvent{
				LocalPort:     localPort,
				RemotePort:    remotePort,
				PodName:       podName,
				ContainerName: containerName,
				Namespace:     namespace,
				PortName:      portName,
				ResourceType:  resourceType,
				ResourceName:  resourceName,
			},
		},
	})
}

func (ev *eventHandler) setState(state proto.State) {
	ev.stateLock.Lock()
	ev.state = state
	ev.stateLock.Unlock()
}

func (ev *eventHandler) handleDeployEvent(e *proto.DeployEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_DeployEvent{
			DeployEvent: e,
		},
	})
}

func (ev *eventHandler) handleStatusCheckEvent(e *proto.StatusCheckEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_StatusCheckEvent{
			StatusCheckEvent: e,
		},
	})
}

func (ev *eventHandler) handleResourceStatusCheckEvent(e *proto.ResourceStatusCheckEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_ResourceStatusCheckEvent{
			ResourceStatusCheckEvent: e,
		},
	})
}

func (ev *eventHandler) handleBuildEvent(e *proto.BuildEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_BuildEvent{
			BuildEvent: e,
		},
	})
}

func LogSkaffoldMetadata(info *version.Info) {
	handler.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Event: &proto.Event{
			EventType: &proto.Event_MetaEvent{
				MetaEvent: &proto.MetaEvent{
					Entry: fmt.Sprintf("Starting Skaffold: %+v", info),
				},
			},
		},
	})
}

func (ev *eventHandler) handle(event *proto.Event) {
	logEntry := &proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Event:     event,
	}

	switch e := event.GetEventType().(type) {
	case *proto.Event_BuildEvent:
		be := e.BuildEvent
		ev.stateLock.Lock()
		ev.state.BuildState.Artifacts[be.Artifact] = be.Status
		ev.stateLock.Unlock()
		switch be.Status {
		case InProgress:
			logEntry.Entry = fmt.Sprintf("Build started for artifact %s", be.Artifact)
		case Complete:
			logEntry.Entry = fmt.Sprintf("Build completed for artifact %s", be.Artifact)
		case Failed:
			logEntry.Entry = fmt.Sprintf("Build failed for artifact %s", be.Artifact)
			// logEntry.Err = be.Err
		default:
		}
	case *proto.Event_DeployEvent:
		de := e.DeployEvent
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
		switch de.Status {
		case InProgress:
			logEntry.Entry = "Deploy started"
		case Complete:
			logEntry.Entry = "Deploy complete"
		case Failed:
			logEntry.Entry = "Deploy failed"
			// logEntry.Err = de.Err
		default:
		}
	case *proto.Event_PortEvent:
		pe := e.PortEvent
		ev.stateLock.Lock()
		ev.state.ForwardedPorts[pe.LocalPort] = pe
		ev.stateLock.Unlock()
		logEntry.Entry = fmt.Sprintf("Forwarding container %s to local port %d", pe.ContainerName, pe.LocalPort)
	case *proto.Event_StatusCheckEvent:
		se := e.StatusCheckEvent
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Status = se.Status
		ev.stateLock.Unlock()
		switch se.Status {
		case Started:
			logEntry.Entry = "Status check started"
		case InProgress:
			logEntry.Entry = "Status check in progress"
		case Succeeded:
			logEntry.Entry = "Status check succeeded"
		case Failed:
			logEntry.Entry = "Status check failed"
		default:
		}
	case *proto.Event_ResourceStatusCheckEvent:
		rse := e.ResourceStatusCheckEvent
		rseName := rse.Resource
		ev.stateLock.Lock()
		ev.state.StatusCheckState.Resources[rseName] = rse.Status
		ev.stateLock.Unlock()
		switch rse.Status {
		case InProgress:
			logEntry.Entry = fmt.Sprintf("Resource %s status updated to %s", rseName, rse.Status)
		case Succeeded:
			logEntry.Entry = fmt.Sprintf("Resource %s status completed successfully", rseName)
		case Failed:
			logEntry.Entry = fmt.Sprintf("Resource %s status failed with %s", rseName, rse.Err)
		default:
		}

	default:
		return
	}

	ev.logEvent(*logEntry)
}

// ResetStateOnBuild resets the build, deploy and sync state
func ResetStateOnBuild() {
	builds := map[string]string{}
	for k := range handler.getState().BuildState.Artifacts {
		builds[k] = NotStarted
	}
	newState := emptyStateWithArtifacts(builds)
	handler.setState(newState)
}

// ResetStateOnDeploy resets the deploy, sync and status check state
func ResetStateOnDeploy() {
	newState := handler.getState()
	newState.DeployState.Status = NotStarted
	newState.StatusCheckState.Status = NotStarted
	newState.ForwardedPorts = map[int32]*proto.PortEvent{}
	handler.setState(newState)
}
