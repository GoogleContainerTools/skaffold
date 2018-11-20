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

// InitializeState instantiates the global state of the skaffold runner, as well as the event log
// This function can only be called once.
func InitializeState(build *latest.BuildConfig, deploy *latest.DeployConfig, port string) error {
	var err error
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
		if err = newStatusServer(port); err != nil {
			err = errors.Wrap(err, "creating status server")
		}
		ev = &eventer{
			eventHandler: handler,
		}
	})
	return err
}

// HandleBuildEvent translates an artifact/status pair into a build logEntry,
// logs it to the eventLog, and updates the global state.
func HandleBuildEvent(artifact string, status string) {
	HandleBuildEventWithError(artifact, status, nil)
}

// HandleBuildEventWithError translates an artifact/status/error tuple into
// a build logEntry, logs it to the eventLog, and updates the global state.
func HandleBuildEventWithError(artifact string, status string, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	var entry string
	switch status {
	case InProgress:
		entry = fmt.Sprintf("Build started for artifact %s", artifact)
	case Complete:
		entry = fmt.Sprintf("Build completed for artifact %s", artifact)
	case Failed:
		entry = fmt.Sprintf("Build failed for artifact %s", artifact)
	default:
	}

	ev.eventHandler.state.BuildState.Artifacts[artifact] = status
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      proto.EventType_buildEvent,
		Entry:     entry,
		Error:     errMsg,
	})
}

// HandleDeployEvent translates a status update to a deploy logEntry,
// logs it to the eventLog, and updates the global state.
func HandleDeployEvent(status string) {
	HandleDeployEventWithError(status, nil)
}

// HandleDeployEventWithError translates a status update to a deploy logEntry with
// the provided error, logs it to the eventLog, and updates the global state.
func HandleDeployEventWithError(status string, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	var entry string
	switch status {
	case InProgress:
		entry = "Deploy started"
	case Complete:
		entry = "Deploy complete"
	case Failed:
		entry = "Deploy failed"
	default:
	}

	ev.eventHandler.state.DeployState.Status = status
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      proto.EventType_deployEvent,
		Entry:     entry,
		Error:     errMsg,
	})
}

func LogSkaffoldMetadata(info *version.Info) {
	ev.logEvent(proto.LogEntry{
		Timestamp: ptypes.TimestampNow(),
		Type:      proto.EventType_metaEvent,
		Entry:     fmt.Sprintf("Starting Skaffold: %+v", info),
	})
}
