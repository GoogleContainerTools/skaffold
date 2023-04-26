/*
Copyright 2023 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
)

// CustomActionTaskInProgress adds an event to mark a custom action task start.
func CustomActionTaskInProgress(name string) {
	handler.handleCustomActionTaskSubtaskEvent(&proto.ExecSubtaskEvent{
		Id:     name,
		TaskId: fmt.Sprintf("%s-%d", constants.Exec, handler.iteration),
		Status: InProgress,
	})
}

// CustomActionTaskFailed adds an event to mark a custom action task has been failed.
func CustomActionTaskFailed(name string, err error) {
	handler.handleCustomActionTaskSubtaskEvent(&proto.ExecSubtaskEvent{
		Id:            name,
		TaskId:        fmt.Sprintf("%s-%d", constants.Exec, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Exec, err),
	})
}

// CustomActionTaskSucceeded adds an event to mark a custom action task has been succeeded.
func CustomActionTaskSucceeded(name string) {
	handler.handleCustomActionTaskSubtaskEvent(&proto.ExecSubtaskEvent{
		Id:     name,
		TaskId: fmt.Sprintf("%s-%d", constants.Exec, handler.iteration),
		Status: Succeeded,
	})
}

func (ev *eventHandler) handleCustomActionTaskSubtaskEvent(e *proto.ExecSubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_ExecEvent{
			ExecEvent: e,
		},
	})
}
