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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	//nolint:golint,staticcheck
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
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
		eventChan: make(chan firedEvent),
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
	eventLog []proto.LogEntry
	logLock  sync.Mutex

	state     proto.State
	stateLock sync.Mutex
	eventChan chan firedEvent
	listeners []*listener
}

type firedEvent struct {
	event *proto.Event
	ts    *timestamp.Timestamp
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

func emptyStatusCheckState() *proto.StatusCheckState {
	return &proto.StatusCheckState{
		Status:     NotStarted,
		Resources:  map[string]string{},
		StatusCode: proto.StatusCode_OK,
	}
}

func TaskInProgress(taskName string, iteration int) {
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:        fmt.Sprintf("%s-%d", taskName, iteration),
		Task:      taskName,
		Iteration: int32(iteration),
		Status:    InProgress,
	})
}

func TaskFailed(taskName sErrors.Phase, iteration int, err error) {
	ae := sErrors.ActionableErrV2(taskName, err)
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:            fmt.Sprintf("%s-%d", taskName, iteration),
		Task:          string(taskName),
		Iteration:     int32(iteration),
		Status:        Failed,
		ActionableErr: ae,
	})
}

func (ev *eventHandler) handle(event *proto.Event) {
	go func(t *timestamp.Timestamp) {
		ev.eventChan <- firedEvent{
			event: event,
			ts:    t,
		}
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

func (ev *eventHandler) handleExec(f firedEvent) {
	logEntry := &proto.LogEntry{
		Timestamp: f.ts,
		Event:     f.event,
	}

	switch e := f.event.GetEventType().(type) {
	case *proto.Event_BuildSubtaskEvent:
		be := e.BuildSubtaskEvent
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
		default:
		}
	case *proto.Event_DeploySubtaskEvent:
		de := e.DeploySubtaskEvent
		ev.stateLock.Lock()
		ev.state.DeployState.Status = de.Status
		ev.stateLock.Unlock()
		switch de.Status {
		case InProgress:
			logEntry.Entry = "Deploy started"
		case Complete:
			logEntry.Entry = "Deploy completed"
		case Failed:
			logEntry.Entry = "Deploy failed"
		default:
		}
	case *proto.Event_PortEvent:
		pe := e.PortEvent
		ev.stateLock.Lock()
		ev.state.ForwardedPorts[pe.LocalPort] = pe
		ev.stateLock.Unlock()
		logEntry.Entry = fmt.Sprintf("Forwarding container %s to local port %d", pe.ContainerName, pe.LocalPort)
	case *proto.Event_StatusCheckSubtaskEvent:
		se := e.StatusCheckSubtaskEvent
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
	case *proto.Event_TaskEvent:
		de := e.TaskEvent
		switch de.Status {
		case InProgress:
			logEntry.Entry = fmt.Sprintf("Task %s initiated", de.Id)
		case Succeeded:
			logEntry.Entry = fmt.Sprintf("Task %s succeeded", de.Id)
		case Failed:
			logEntry.Entry = fmt.Sprintf("Task %s failed with error code %v", de.Id, de.ActionableErr.ErrCode)
		}
	}
	ev.logEvent(*logEntry)
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
