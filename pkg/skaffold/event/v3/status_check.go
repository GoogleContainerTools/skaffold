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
	protoV3 "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func ResourceStatusCheckEventCompleted(r string, ae *protoV3.ActionableErr) {
	if ae.ErrCode != protoV3.StatusCode_STATUSCHECK_SUCCESS {
		resourceStatusCheckEventFailed(r, ae)
		return
	}
	resourceStatusCheckEventSucceeded(r)
}

func ResourceStatusCheckEventCompletedMessage(r string, message string, ae *protoV3.ActionableErr) {
	ResourceStatusCheckEventCompleted(r, ae)
	event := &protoV3.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   message,
		Level:     -1,
	}
	handler.handle(event, SkaffoldLogEvent)
}

func resourceStatusCheckEventSucceeded(r string) {
	event := &protoV3.StatusCheckSucceededEvent{
		Id:         r,
		TaskId:     fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:   r,
		Status:     Succeeded,
		Message:    Succeeded,
		StatusCode: protoV3.StatusCode_STATUSCHECK_SUCCESS,
	}
	updateStatusCheckState(event.Resource, event.Status)
	handler.handle(event, StatusCheckSucceededEvent)
}

func resourceStatusCheckEventFailed(r string, ae *protoV3.ActionableErr) {
	event := &protoV3.StatusCheckFailedEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        Failed,
		StatusCode:    ae.ErrCode,
		ActionableErr: ae,
	}
	updateStatusCheckState(event.Resource, event.Status)
	handler.handle(event, StatusCheckFailedEvent)
}

func ResourceStatusCheckEventUpdated(r string, ae *protoV3.ActionableErr) {
	event := &protoV3.StatusCheckStartedEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Status:        InProgress,
		Message:       ae.Message,
		StatusCode:    ae.ErrCode,
		ActionableErr: ae,
	}
	updateStatusCheckState(event.Resource, event.Status)
	handler.handle(event, StatusCheckStartedEvent)
}

func ResourceStatusCheckEventUpdatedMessage(r string, message string, ae *protoV3.ActionableErr) {
	ResourceStatusCheckEventUpdated(r, ae)
	event := &protoV3.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		SubtaskId: r,
		Message:   fmt.Sprintf("%s %s\n", message, ae.Message),
		Level:     -1,
	}
	handler.handle(event, SkaffoldLogEvent)
}

func updateStatusCheckState(resrouce string, status string) {
	handler.stateLock.Lock()
	handler.state.StatusCheckState.Resources[resrouce] = status
	handler.stateLock.Unlock()
}
