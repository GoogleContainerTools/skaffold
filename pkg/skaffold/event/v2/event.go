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

//nolint:golint,staticcheck

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/acarl005/stripansi"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/go-homedir"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/enums"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
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
		wait:      make(chan bool, 1),
		state:     &proto.State{},
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
	eventLog            []*proto.Event
	logLock             sync.Mutex
	applicationLogs     []*proto.Event
	applicationLogsLock sync.Mutex
	cfg                 Config

	iteration               int
	errorOnce               sync.Once
	wait                    chan bool
	state                   *proto.State
	stateLock               sync.Mutex
	eventChan               chan *proto.Event
	eventListeners          []*listener
	applicationLogListeners []*listener
}

type listener struct {
	callback func(*proto.Event) error
	errors   chan error
	closed   bool
}

func GetIteration() int {
	return handler.iteration
}

func ForEachEvent(callback func(*proto.Event) error) error {
	return handler.forEachEvent(callback)
}

func ForEachApplicationLog(callback func(*proto.Event) error) error {
	return handler.forEachApplicationLog(callback)
}

func (ev *eventHandler) forEachEvent(callback func(*proto.Event) error) error {
	// Unblock call to `WaitForConnection()`
	select {
	case handler.wait <- true:
	default:
	}
	return ev.forEach(&ev.eventListeners, &ev.eventLog, &ev.logLock, callback)
}

func (ev *eventHandler) forEachApplicationLog(callback func(*proto.Event) error) error {
	return ev.forEach(&ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock, callback)
}

func (ev *eventHandler) forEach(listeners *[]*listener, log *[]*proto.Event, lock sync.Locker, callback func(*proto.Event) error) error {
	listener := &listener{
		callback: callback,
		errors:   make(chan error),
	}

	lock.Lock()

	oldEvents := make([]*proto.Event, len(*log))
	copy(oldEvents, *log)
	*listeners = append(*listeners, listener)

	lock.Unlock()

	for i := range oldEvents {
		if err := callback(oldEvents[i]); err != nil {
			// listener should maybe be closed
			return err
		}
	}

	return <-listener.errors
}

func Handle(event *proto.Event) error {
	if event != nil {
		handler.handle(event)
	}
	return nil
}

// WaitForConnection will block execution until the server receives a connection
func WaitForConnection() {
	<-handler.wait
}

func (ev *eventHandler) logEvent(event *proto.Event) {
	ev.log(event, &ev.eventListeners, &ev.eventLog, &ev.logLock)
}

func (ev *eventHandler) logApplicationLog(event *proto.Event) {
	ev.log(event, &ev.applicationLogListeners, &ev.applicationLogs, &ev.applicationLogsLock)
}

func (ev *eventHandler) log(event *proto.Event, listeners *[]*listener, log *[]*proto.Event, lock sync.Locker) {
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
	*log = append(*log, event)

	lock.Unlock()
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

// SendErrorMessageOnce sends an error message to skaffold log events stream only once.
// Use it if you want to avoid sending duplicate error messages.
func SendErrorMessageOnce(task constants.Phase, subtaskID string, err error) {
	handler.sendErrorMessage(task, subtaskID, err)
}

func (ev *eventHandler) sendErrorMessage(task constants.Phase, subtask string, err error) {
	if err == nil {
		return
	}

	ev.errorOnce.Do(func() {
		ev.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
			TaskId:    fmt.Sprintf("%s-%d", task, handler.iteration),
			SubtaskId: subtask,
			Message:   fmt.Sprintf("%s\n", err),
			Level:     enums.LogLevel_STANDARD,
		})
	})
}

func (ev *eventHandler) handle(event *proto.Event) {
	event.Timestamp = timestamppb.Now()
	ev.eventChan <- event
	if _, ok := event.GetEventType().(*proto.Event_TerminationEvent); ok {
		// close the event channel indicating there are no more events to all the receivers
		close(ev.eventChan)
	}
}

func (ev *eventHandler) handleExec(event *proto.Event) {
	switch e := event.GetEventType().(type) {
	case *proto.Event_ApplicationLogEvent:
		ev.logApplicationLog(event)
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
	case *proto.Event_RenderEvent:
		te := e.RenderEvent
		ev.stateLock.Lock()
		ev.state.RenderState.Status = te.Status
		ev.stateLock.Unlock()
	case *proto.Event_VerifyEvent:
		te := e.VerifyEvent
		ev.stateLock.Lock()
		ev.state.VerifyState.Status = te.Status
		ev.stateLock.Unlock()
	case *proto.Event_ExecEvent:
		te := e.ExecEvent
		ev.stateLock.Lock()
		ev.state.ExecState.Status = te.Status
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
	// Ensure that the filepath provided has the directories available when attemping to save the file.
	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("unable to create directory %q: %w", dir, err)
	}
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", fp, err)
	}
	defer f.Close()
	marshaller := jsonpb.Marshaler{}
	for _, ev := range handler.eventLog {
		contents := bytes.NewBuffer([]byte{})
		if err := marshaller.Marshal(contents, ev); err != nil {
			return fmt.Errorf("marshalling event: %w", err)
		}
		if _, err := f.WriteString(contents.String() + "\n"); err != nil {
			return fmt.Errorf("writing string: %w", err)
		}
	}
	handler.logLock.Unlock()
	return nil
}

// SaveLastLog writes the output from the previous run to the specified filepath
func SaveLastLog(fp string) error {
	handler.logLock.Lock()
	defer handler.logLock.Unlock()

	// Create file to write logs to
	fp, err := lastLogFile(fp)
	if err != nil {
		return fmt.Errorf("getting last log file %w", err)
	}
	// Ensure that the filepath provided has the directories available when attemping to save the file.
	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("unable to create directory %q: %w", dir, err)
	}
	f, err := os.OpenFile(fp, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", fp, err)
	}
	defer f.Close()

	// Iterate over events, grabbing contents only from SkaffoldLogEvents
	var contents bytes.Buffer
	for _, ev := range handler.eventLog {
		if sle := ev.GetSkaffoldLogEvent(); sle != nil {
			// Strip ansi color sequences as this makes it easier to deal with when pasting into github issues
			if _, err = contents.WriteString(stripansi.Strip(sle.Message)); err != nil {
				return fmt.Errorf("writing string to temporary buffer: %w", err)
			}
		}
	}

	// Write contents of temporary buffer to file
	if _, err = f.Write(contents.Bytes()); err != nil {
		return fmt.Errorf("writing buffer contents to file: %w", err)
	}
	return nil
}

func lastLogFile(fp string) (string, error) {
	if fp != "" {
		return fp, nil
	}

	// last log location unspecified, use ~/.skaffold/last.log
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("retrieving home directory: %w", err)
	}
	return filepath.Join(home, constants.DefaultSkaffoldDir, "last.log"), nil
}
