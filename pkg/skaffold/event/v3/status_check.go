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

package v3

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
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
	event := &proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   message,
		Level:     -1,
	}
	WrapInMainAndHandle(event.TaskId, event, SkaffoldLogEvent)
}

func resourceStatusCheckEventSucceeded(r string) {
	event := &proto.StatusCheckSucceededEvent{
		Id:         r,
		TaskId:     fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:   r,
		Status:     Succeeded,
		Message:    Succeeded,
		StatusCode: proto.StatusCode_STATUSCHECK_SUCCESS,
	}
	WrapInMainAndHandle(r, event, StatusCheckSucceededEvent)
}

func resourceStatusCheckEventFailed(r string, ae proto.ActionableErr) {
	event := &proto.StatusCheckFailedEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        Failed,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	}
	WrapInMainAndHandle(r, event, StatusCheckStartedEvent)
}

func ResourceStatusCheckEventUpdated(r string, ae proto.ActionableErr) {
	event := &proto.StatusCheckStartedEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        InProgress,
		Message:       ae.Message,
		StatusCode:    ae.ErrCode,
		ActionableErr: &ae,
	}
	WrapInMainAndHandle(r, event, StatusCheckStartedEvent)
}

func ResourceStatusCheckEventUpdatedMessage(r string, message string, ae proto.ActionableErr) {
	ResourceStatusCheckEventUpdated(r, ae)
	event := &proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   fmt.Sprintf("%s %s\n", message, ae.Message),
		Level:     -1,
	}
	WrapInMainAndHandle(event.TaskId, event, SkaffoldLogEvent)
}
