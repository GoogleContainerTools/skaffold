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
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	NotStarted = "Not Started"
	InProgress = "In Progress"
	Complete   = "Complete"
	Failed     = "Failed"
)

var (
	handler    *eventHandler
	once       sync.Once
	pluginMode bool

	cli proto.SkaffoldServiceClient // for plugin RPC connections
)

type eventHandler struct {
	eventLog []proto.LogEntry
	logLock  sync.Mutex

	state     proto.State
	stateLock sync.Mutex

	listeners []listener
}

type listener struct {
	callback func(*proto.LogEntry) error
	errors   chan error
	closed   bool
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
	listener := listener{
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

func emptyState(build *latest.BuildConfig) proto.State {
	builds := map[string]string{}
	if build != nil {
		for _, a := range build.Artifacts {
			builds[a.ImageName] = NotStarted
		}
	}

	return proto.State{
		BuildState: &proto.BuildState{
			Artifacts: builds,
		},
		DeployState: &proto.DeployState{
			Status: NotStarted,
		},
		ForwardedPorts: make(map[string]*proto.PortEvent),
	}
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
// It returns a shutdown callback for tearing down the grpc server, which the runner is responsible for calling.
// This function can only be called once.
func InitializeState(build *latest.BuildConfig, deploy *latest.DeployConfig, opts *config.SkaffoldOptions) (func() error, error) {
	var err error
	serverShutdown := func() error { return nil }
	once.Do(func() {
		handler = &eventHandler{
			state: emptyState(build),
		}

		if opts.EnableRPC {
			serverShutdown, err = newStatusServer(opts.RPCPort, opts.RPCHTTPPort)
			if err != nil {
				err = errors.Wrap(err, "creating status server")
				return
			}
		}
	})
	return serverShutdown, err
}

func SetupRPCClient(opts *config.SkaffoldOptions) error {
	pluginMode = true
	conn, err := grpc.Dial(fmt.Sprintf(":%d", opts.RPCPort), grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "opening gRPC connection to remote skaffold process")
	}
	cli = proto.NewSkaffoldServiceClient(conn)
	return nil
}

// DeployInProgress notifies that a deployment has been started.
func DeployInProgress() {
	handler.handleDeployEvent(&proto.DeployEvent{Status: InProgress})
}

// DeployFailed notifies that a deployment has failed.
func DeployFailed(err error) {
	handler.handleDeployEvent(&proto.DeployEvent{Status: Failed, Err: err.Error()})
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
func PortForwarded(localPort, remotePort int32, podName, containerName, namespace string) {
	handler.doHandle(&proto.Event{
		EventType: &proto.Event_PortEvent{
			PortEvent: &proto.PortEvent{
				LocalPort:     localPort,
				RemotePort:    remotePort,
				PodName:       podName,
				ContainerName: containerName,
				Namespace:     namespace,
			},
		},
	})
}

func (ev *eventHandler) handleDeployEvent(e *proto.DeployEvent) {
	ev.doHandle(&proto.Event{
		EventType: &proto.Event_DeployEvent{
			DeployEvent: e,
		},
	})
}

func (ev *eventHandler) handleBuildEvent(e *proto.BuildEvent) {
	ev.doHandle(&proto.Event{
		EventType: &proto.Event_BuildEvent{
			BuildEvent: e,
		},
	})
}

func (ev *eventHandler) doHandle(event *proto.Event) {
	if pluginMode {
		go cli.Handle(context.Background(), event)
	} else {
		go ev.handle(event)
	}
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
		ev.state.ForwardedPorts[pe.ContainerName] = pe
		ev.stateLock.Unlock()
		logEntry.Entry = fmt.Sprintf("Forwarding container %s to local port %d", pe.ContainerName, pe.LocalPort)
	default:
		return
	}

	ev.logEvent(*logEntry)
}
