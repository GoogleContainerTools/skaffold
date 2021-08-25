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
	"github.com/GoogleContainerTools/skaffold/proto/enums"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func ResourceStatusCheckEventCompleted(r string, ae proto.ActionableErr) {
	if ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		resourceStatusCheckEventFailed(r, ae)
		return
	}
	resourceStatusCheckEventSucceeded(r)
}

func ResourceStatusCheckEventCompletedMessage(r string, message string, ae proto.ActionableErr) {
	ResourceStatusCheckEventCompleted(r, ae)
	handler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   message,
		Level:     enums.LogLevel_STANDARD,
	})
}

func resourceStatusCheckEventSucceeded(r string) {
	handler.handleStatusCheckSubtaskEvent(&proto.StatusCheckSubtaskEvent{
		Id:         r,
		TaskId:     fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:   r,
		Status:     Succeeded,
		Message:    Succeeded,
		StatusCode: proto.StatusCode_STATUSCHECK_SUCCESS,
	})
}

func resourceStatusCheckEventFailed(r string, ae proto.ActionableErr) {
	handler.handleStatusCheckSubtaskEvent(&proto.StatusCheckSubtaskEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        Failed,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	})
}

func ResourceStatusCheckEventUpdated(r string, ae proto.ActionableErr) {
	handler.handleStatusCheckSubtaskEvent(&proto.StatusCheckSubtaskEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        InProgress,
		Message:       ae.Message,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	})
}

func ResourceStatusCheckEventUpdatedMessage(r string, message string, ae proto.ActionableErr) {
	ResourceStatusCheckEventUpdated(r, ae)
	handler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   fmt.Sprintf("%s %s\n", message, ae.Message),
		Level:     enums.LogLevel_STANDARD,
	})
}

func (ev *eventHandler) handleStatusCheckSubtaskEvent(e *proto.StatusCheckSubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_StatusCheckSubtaskEvent{
			StatusCheckSubtaskEvent: e,
		},
	})
}
