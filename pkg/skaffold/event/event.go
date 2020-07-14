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

	"github.com/golang/protobuf/ptypes"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/proto"
)

const (
	NotStarted = "Not Started"
	InProgress = "In Progress"
	Complete   = "Complete"
	Failed     = "Failed"
	Info       = "Information"
	Started    = "Started"
	Succeeded  = "Succeeded"
	Terminated = "Terminated"
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

func emptyState(p latest.Pipeline, kubeContext string, autoBuild, autoDeploy, autoSync bool) proto.State {
	builds := map[string]string{}
	for _, a := range p.Build.Artifacts {
		builds[a.ImageName] = NotStarted
	}
	metadata := initializeMetadata(p, kubeContext)
	return emptyStateWithArtifacts(builds, metadata, autoBuild, autoDeploy, autoSync)
}

func emptyStateWithArtifacts(builds map[string]string, metadata *proto.Metadata, autoBuild, autoDeploy, autoSync bool) proto.State {
	return proto.State{
		BuildState: &proto.BuildState{
			Artifacts:   builds,
			AutoTrigger: autoBuild,
			StatusCode:  proto.StatusCode_OK,
		},
		DeployState: &proto.DeployState{
			Status:      NotStarted,
			AutoTrigger: autoDeploy,
			StatusCode:  proto.StatusCode_OK,
		},
		StatusCheckState: emptyStatusCheckState(),
		ForwardedPorts:   make(map[int32]*proto.PortEvent),
		FileSyncState: &proto.FileSyncState{
			Status:      NotStarted,
			AutoTrigger: autoSync,
		},
		Metadata: metadata,
	}
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
func InitializeState(c latest.Pipeline, kc string, autoBuild, autoDeploy, autoSync bool) {
	handler.setState(emptyState(c, kc, autoBuild, autoDeploy, autoSync))
}

// DeployInProgress notifies that a deployment has been started.
func DeployInProgress() {
	handler.handleDeployEvent(&proto.DeployEvent{Status: InProgress})
}

// DeployFailed notifies that non-fatal errors were encountered during a deployment.
func DeployFailed(err error) {
	aiErr := sErrors.ActionableErr(sErrors.Deploy, err)
	handler.stateLock.Lock()
	handler.state.DeployState.StatusCode = aiErr.ErrCode
	handler.stateLock.Unlock()
	handler.handleDeployEvent(&proto.DeployEvent{Status: Failed,
		Err:           err.Error(),
		ErrCode:       aiErr.ErrCode,
		ActionableErr: aiErr})
}

// DeployEvent notifies that a deployment of non fatal interesting errors during deploy.
func DeployInfoEvent(err error) {
	handler.handleDeployEvent(&proto.DeployEvent{Status: Info, Err: err.Error()})
}

func StatusCheckEventEnded(errCode proto.StatusCode, err error) {
	if err != nil {
		handler.stateLock.Lock()
		handler.state.StatusCheckState.StatusCode = errCode
		handler.stateLock.Unlock()
		statusCheckEventFailed(err, errCode)
		return
	}
	handler.stateLock.Lock()
	handler.state.StatusCheckState.StatusCode = proto.StatusCode_STATUSCHECK_SUCCESS
	handler.stateLock.Unlock()
	statusCheckEventSucceeded()
}

func statusCheckEventSucceeded() {
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status: Succeeded,
	})
}

func statusCheckEventFailed(err error, errCode proto.StatusCode) {
	aiErr := &proto.ActionableErr{
		ErrCode: errCode,
		Message: err.Error(),
	}
	handler.handleStatusCheckEvent(&proto.StatusCheckEvent{
		Status:        Failed,
		Err:           err.Error(),
		ErrCode:       errCode,
		ActionableErr: aiErr})
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

func ResourceStatusCheckEventCompleted(r string, ae proto.ActionableErr) {
	if ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		resourceStatusCheckEventFailed(r, ae)
		return
	}
	resourceStatusCheckEventSucceeded(r)
}

func resourceStatusCheckEventSucceeded(r string) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource:   r,
		Status:     Succeeded,
		Message:    Succeeded,
		StatusCode: proto.StatusCode_STATUSCHECK_SUCCESS,
	})
}

func resourceStatusCheckEventFailed(r string, ae proto.ActionableErr) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource:      r,
		Status:        Failed,
		Err:           ae.Message,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	})
}

func ResourceStatusCheckEventUpdated(r string, ae proto.ActionableErr) {
	handler.handleResourceStatusCheckEvent(&proto.ResourceStatusCheckEvent{
		Resource:      r,
		Status:        InProgress,
		Message:       ae.Message,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	})
}

// DeployComplete notifies that a deployment has completed.
func DeployComplete() {
	handler.stateLock.Lock()
	handler.state.DeployState.StatusCode = proto.StatusCode_DEPLOY_SUCCESS
	handler.stateLock.Unlock()
	handler.handleDeployEvent(&proto.DeployEvent{Status: Complete})
}

// BuildInProgress notifies that a build has been started.
func BuildInProgress(imageName string) {
	handler.handleBuildEvent(&proto.BuildEvent{Artifact: imageName, Status: InProgress})
}

// BuildFailed notifies that a build has failed.
func BuildFailed(imageName string, err error) {
	aiErr := sErrors.ActionableErr(sErrors.Build, err)
	handler.stateLock.Lock()
	handler.state.BuildState.StatusCode = aiErr.ErrCode
	handler.stateLock.Unlock()
	handler.handleBuildEvent(&proto.BuildEvent{
		Artifact:      imageName,
		Status:        Failed,
		Err:           err.Error(),
		ErrCode:       aiErr.ErrCode,
		ActionableErr: aiErr})
}

// BuildComplete notifies that a build has completed.
func BuildComplete(imageName string) {
	handler.handleBuildEvent(&proto.BuildEvent{Artifact: imageName, Status: Complete})
}

// DevLoopInProgress notifies that a dev loop has been started.
func DevLoopInProgress(i int) {
	handler.handleDevLoopEvent(&proto.DevLoopEvent{Iteration: int32(i), Status: InProgress})
}

// DevLoopFailed notifies that a dev loop has failed with an error code
func DevLoopFailedWithErrorCode(i int, statusCode proto.StatusCode, err error) {
	ai := &proto.ActionableErr{
		ErrCode: statusCode,
		Message: err.Error(),
	}
	handler.handleDevLoopEvent(&proto.DevLoopEvent{
		Iteration: int32(i),
		Status:    Failed,
		Err:       ai})
}

// DevLoopFailed notifies that a dev loop has failed in a given phase
func DevLoopFailedInPhase(iteration int, phase sErrors.Phase, err error) {
	state := handler.getState()
	switch phase {
	case sErrors.Deploy:
		if state.DeployState.StatusCode != proto.StatusCode_DEPLOY_SUCCESS {
			DevLoopFailedWithErrorCode(iteration, state.DeployState.StatusCode, err)
		} else {
			DevLoopFailedWithErrorCode(iteration, state.StatusCheckState.StatusCode, err)
		}
	case sErrors.StatusCheck:
		DevLoopFailedWithErrorCode(iteration, state.StatusCheckState.StatusCode, err)
	case sErrors.Build:
		DevLoopFailedWithErrorCode(iteration, state.BuildState.StatusCode, err)
	default:
		ai := sErrors.ActionableErr(phase, err)
		DevLoopFailedWithErrorCode(iteration, ai.ErrCode, err)
	}
}

// DevLoopComplete notifies that a dev loop has completed.
func DevLoopComplete(i int) {
	handler.handleDevLoopEvent(&proto.DevLoopEvent{Iteration: int32(i), Status: Succeeded})
}

// FileSyncInProgress notifies that a file sync has been started.
func FileSyncInProgress(fileCount int, image string) {
	handler.handleFileSyncEvent(&proto.FileSyncEvent{FileCount: int32(fileCount), Image: image, Status: InProgress})
}

// FileSyncFailed notifies that a file sync has failed.
func FileSyncFailed(fileCount int, image string, err error) {
	aiErr := sErrors.ActionableErr(sErrors.FileSync, err)
	handler.handleFileSyncEvent(&proto.FileSyncEvent{FileCount: int32(fileCount), Image: image, Status: Failed,
		Err: err.Error(), ErrCode: aiErr.ErrCode, ActionableErr: aiErr})
}

// FileSyncSucceeded notifies that a file sync has succeeded.
func FileSyncSucceeded(fileCount int, image string) {
	handler.handleFileSyncEvent(&proto.FileSyncEvent{FileCount: int32(fileCount), Image: image, Status: Succeeded})
}

// PortForwarded notifies that a remote port has been forwarded locally.
func PortForwarded(localPort, remotePort int32, podName, containerName, namespace string, portName string, resourceType, resourceName, address string) {
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
				Address:       address,
			},
		},
	})
}

// DebuggingContainerStarted notifies that a debuggable container has appeared.
func DebuggingContainerStarted(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	go handler.handle(&proto.Event{
		EventType: &proto.Event_DebuggingContainerEvent{
			DebuggingContainerEvent: &proto.DebuggingContainerEvent{
				Status:        Started,
				PodName:       podName,
				ContainerName: containerName,
				Namespace:     namespace,
				Artifact:      artifact,
				Runtime:       runtime,
				WorkingDir:    workingDir,
				DebugPorts:    debugPorts,
			},
		},
	})
}

// DebuggingContainerTerminated notifies that a debuggable container has disappeared.
func DebuggingContainerTerminated(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	go handler.handle(&proto.Event{
		EventType: &proto.Event_DebuggingContainerEvent{
			DebuggingContainerEvent: &proto.DebuggingContainerEvent{
				Status:        Terminated,
				PodName:       podName,
				ContainerName: containerName,
				Namespace:     namespace,
				Artifact:      artifact,
				Runtime:       runtime,
				WorkingDir:    workingDir,
				DebugPorts:    debugPorts,
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

func (ev *eventHandler) handleDevLoopEvent(e *proto.DevLoopEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_DevLoopEvent{
			DevLoopEvent: e,
		},
	})
}

func (ev *eventHandler) handleFileSyncEvent(e *proto.FileSyncEvent) {
	go ev.handle(&proto.Event{
		EventType: &proto.Event_FileSyncEvent{
			FileSyncEvent: e,
		},
	})
}

func LogMetaEvent() {
	metadata := handler.state.Metadata
	handler.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Event: &proto.Event{
			EventType: &proto.Event_MetaEvent{
				MetaEvent: &proto.MetaEvent{
					Entry:    fmt.Sprintf("Starting Skaffold: %+v", version.Get()),
					Metadata: metadata,
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
	case *proto.Event_FileSyncEvent:
		fse := e.FileSyncEvent
		fseFileCount := fse.FileCount
		fseImage := fse.Image
		ev.stateLock.Lock()
		ev.state.FileSyncState.Status = fse.Status
		ev.stateLock.Unlock()
		switch fse.Status {
		case InProgress:
			logEntry.Entry = fmt.Sprintf("File sync started for %d files for %s", fseFileCount, fseImage)
		case Succeeded:
			logEntry.Entry = fmt.Sprintf("File sync succeeded for %d files for %s", fseFileCount, fseImage)
		case Failed:
			logEntry.Entry = fmt.Sprintf("File sync failed for %d files for %s", fseFileCount, fseImage)
		default:
		}
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
		switch de.Status {
		case Started:
			logEntry.Entry = fmt.Sprintf("Debuggable container started pod/%s:%s (%s)", de.PodName, de.ContainerName, de.Namespace)
		case Terminated:
			logEntry.Entry = fmt.Sprintf("Debuggable container terminated pod/%s:%s (%s)", de.PodName, de.ContainerName, de.Namespace)
		}
	case *proto.Event_DevLoopEvent:
		de := e.DevLoopEvent
		switch de.Status {
		case InProgress:
			logEntry.Entry = fmt.Sprintf("DevInit Iteration %d in progress", de.Iteration)
		case Succeeded:
			logEntry.Entry = fmt.Sprintf("DevInit Iteration %d successful", de.Iteration)
		case Failed:
			logEntry.Entry = fmt.Sprintf("DevInit Iteration %d failed with error code %v", de.Iteration, de.Err.ErrCode)
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
	autoBuild, autoDeploy, autoSync := handler.getState().BuildState.AutoTrigger, handler.getState().DeployState.AutoTrigger, handler.getState().FileSyncState.AutoTrigger
	newState := emptyStateWithArtifacts(builds, handler.getState().Metadata, autoBuild, autoDeploy, autoSync)
	handler.setState(newState)
}

// ResetStateOnDeploy resets the deploy, sync and status check state
func ResetStateOnDeploy() {
	newState := handler.getState()
	newState.DeployState.Status = NotStarted
	newState.DeployState.StatusCode = proto.StatusCode_OK
	newState.StatusCheckState = emptyStatusCheckState()
	newState.ForwardedPorts = map[int32]*proto.PortEvent{}
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

func AutoTriggerDiff(name string, val bool) (bool, error) {
	switch name {
	case "build":
		return val != handler.getState().BuildState.AutoTrigger, nil
	case "sync":
		return val != handler.getState().FileSyncState.AutoTrigger, nil
	case "deploy":
		return val != handler.getState().DeployState.AutoTrigger, nil
	default:
		return false, fmt.Errorf("unknown phase %v not found in handler state", name)
	}
}
