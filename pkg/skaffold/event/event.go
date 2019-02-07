/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"
)

const (
	NotStarted = "Not Started"
	InProgress = "In Progress"
	Complete   = "Complete"
	Failed     = "Failed"
)

var ev *eventer
var once sync.Once

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
func InitializeState(build *latest.BuildConfig, deploy *latest.DeployConfig, portOrSocket string) (func(), error) {
	var err error
	var shutdown func()
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
		}

		handler := &eventHandler{
			eventLog: eventLog{},
			state:    state,
		}
		shutdown, err = newStatusServer(portOrSocket)
		if err != nil {
			err = errors.Wrap(err, "creating status server")
		}
		ev = &eventer{
			eventHandler: handler,
		}
	})
	return shutdown, err
}

func Handle(event Event) {
	var entry string
	if event.EventType == Build {
		ev.eventHandler.state.BuildState.Artifacts[event.Artifact] = event.Status
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
		ev.eventHandler.state.DeployState.Status = event.Status
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

	var errStr string
	if event.Err != nil {
		errStr = event.Err.Error()
	}
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      event.EventType,
		Entry:     entry,
		Error:     errStr,
	})
}

func LogSkaffoldMetadata(info *version.Info) {
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      proto.EventType_metaEvent,
		Entry:     fmt.Sprintf("Starting Skaffold: %+v", info),
	})
}
