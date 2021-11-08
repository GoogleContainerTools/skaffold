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

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TaskInProgress(task constants.Phase, description string) {
	// Special casing to increment iteration and clear application and skaffold logs
	if task == constants.DevLoop {
		handler.iteration++

		handler.applicationLogs = []*proto.Event{}
	}

	handler.handleTaskEvent(&proto.TaskEvent{
		Id:          fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:        string(task),
		Description: description,
		Iteration:   int32(handler.iteration),
		Status:      InProgress,
	})
}

func TaskFailed(task constants.Phase, err error) {
	ae := sErrors.ActionableErrV2(handler.cfg, task, err)
	handler.sendErrorMessage(task, constants.SubtaskIDNone, err)
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:            fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:          string(task),
		Iteration:     int32(handler.iteration),
		Status:        Failed,
		ActionableErr: ae,
	})
}

func TaskSucceeded(task constants.Phase) {
	handler.handleTaskEvent(&proto.TaskEvent{
		Id:        fmt.Sprintf("%s-%d", task, handler.iteration),
		Task:      string(task),
		Iteration: int32(handler.iteration),
		Status:    Succeeded,
	})
}

func (ev *eventHandler) handleTaskEvent(e *proto.TaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_TaskEvent{
			TaskEvent: e,
		},
	})
}
