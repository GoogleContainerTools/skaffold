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
	ev         *eventHandler
	once       sync.Once
	pluginMode bool

	cli proto.SkaffoldServiceClient // for plugin RPC connections
)

type eventHandler struct {
	eventLog []proto.LogEntry
	logLock  sync.Mutex

	state     proto.State
	stateLock sync.Mutex

	listeners []chan proto.LogEntry
}

func (ev *eventHandler) RegisterListener(listener chan proto.LogEntry) {
	ev.listeners = append(ev.listeners, listener)
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

	for _, c := range ev.listeners {
		c <- entry
	}
	ev.eventLog = append(ev.eventLog, entry)

	ev.logLock.Unlock()
}

func (ev *eventHandler) forEachEvent(callback func(*proto.LogEntry) error) error {
	c := make(chan proto.LogEntry)

	ev.logLock.Lock()

	oldEvents := make([]proto.LogEntry, len(ev.eventLog))
	copy(oldEvents, ev.eventLog)
	ev.RegisterListener(c)

	ev.logLock.Unlock()

	for i := range oldEvents {
		if err := callback(&oldEvents[i]); err != nil {
			return err
		}
	}

	for {
		entry := <-c
		if err := callback(&entry); err != nil {
			return err
		}
	}
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
		ev = &eventHandler{
			state: emptyState(build),
		}

		if opts.EnableRPC {
			serverShutdown, err = newStatusServer(opts.RPCPort)
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

func Handle(event *proto.Event) {
	if pluginMode {
		go cli.Handle(context.Background(), event)
	} else {
		go handle(event)
	}
}

func handle(event *proto.Event) {
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

func LogSkaffoldMetadata(info *version.Info) {
	ev.logEvent(proto.LogEntry{
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
