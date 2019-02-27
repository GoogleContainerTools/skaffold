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
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes"
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

type eventLog []proto.LogEntry

type Event struct {
	Artifact  string
	EventType proto.EventType
	Status    string
	Err       error
}

const (
	Build  = proto.EventType_buildEvent
	Deploy = proto.EventType_deployEvent
	Port   = proto.EventType_portEvent
	Meta   = proto.EventType_metaEvent
)

type eventer struct {
	*eventHandler
}

type eventHandler struct {
	eventLog

	listeners []chan proto.LogEntry
	state     *proto.State
}

func (ev *eventHandler) RegisterListener(listener chan proto.LogEntry) {
	ev.listeners = append(ev.listeners, listener)
}

func (ev *eventHandler) logEvent(entry proto.LogEntry) {
	for _, c := range ev.listeners {
		c <- entry
	}
	ev.eventLog = append(ev.eventLog, entry)
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
// It returns a shutdown callback for tearing down the grpc server, which the runner is responsible for calling.
// This function can only be called once.
func InitializeState(build *latest.BuildConfig, deploy *latest.DeployConfig, addr string) (func(), error) {
	var err error
	var serverShutdown func()
	var conn *grpc.ClientConn
	once.Do(func() {
		builds := map[string]string{}
		deploys := map[string]string{}
		if build != nil {
			for _, a := range build.Artifacts {
				builds[a.ImageName] = NotStarted
				deploys[a.ImageName] = NotStarted
			}
		}
		state := &proto.State{
			BuildState: &proto.BuildState{
				Artifacts: builds,
			},
			DeployState: &proto.DeployState{
				Status: NotStarted,
			},
			ForwardedPorts: make(map[string]*proto.PortInfo),
		}
		ev = &eventHandler{
			eventLog: eventLog{},
			state:    state,
		}
		serverShutdown, err = newStatusServer(addr)
		if err != nil {
			err = errors.Wrap(err, "creating status server")
			return
		}
	})
	return func() {
		serverShutdown()
		if conn != nil {
			conn.Close()
		}
	}, err
}

func SetupRPCClient(opts *config.SkaffoldOptions) error {
	pluginMode = true
	conn, err := grpc.Dial(opts.RPCPort, grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "opening gRPC connection to remote skaffold process")
	}
	cli = proto.NewSkaffoldServiceClient(conn)
	return nil
}

func Handle(event proto.Event) {
	if pluginMode {
		go cli.Handle(context.Background(), &event)
	} else {
		go handle(event)
	}
}

func handle(event proto.Event) {
	var entry string
	if event.EventType == Build {
		ev.state.BuildState.Artifacts[event.Artifact] = event.Status
		switch event.Status {
		case InProgress:
			entry = fmt.Sprintf("Build started for artifact %s", event.Artifact)
		case Complete:
			entry = fmt.Sprintf("Build completed for artifact %s", event.Artifact)
		case Failed:
			entry = fmt.Sprintf("Build failed for artifact %s", event.Artifact)
		default:
		}
	}
	if event.EventType == Deploy {
		ev.state.DeployState.Status = event.Status
		switch event.Status {
		case InProgress:
			entry = "Deploy started"
		case Complete:
			entry = "Deploy complete"
		case Failed:
			entry = "Deploy failed"
		default:
		}
	}
	if event.EventType == Port {
		ev.state.ForwardedPorts[event.PortInfo.ContainerName] = event.PortInfo
		entry = fmt.Sprintf("Forwarding container %s to local port %d", event.PortInfo.ContainerName, event.PortInfo.LocalPort)
	}

	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      event.EventType,
		Entry:     entry,
		Error:     event.Err,
	})
}

func LogSkaffoldMetadata(info *version.Info) {
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      proto.EventType_metaEvent,
		Entry:     fmt.Sprintf("Starting Skaffold: %+v", info),
	})
}
