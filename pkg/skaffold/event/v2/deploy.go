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
	"strconv"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
)

func DeployInProgress(id int) {
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status: InProgress,
	})
}

func DeployFailed(id int, err error) {
	handler.sendErrorMessage(constants.Deploy, strconv.Itoa(id), err)
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Deploy, err),
	})
}

func DeploySucceeded(id int) {
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status: Succeeded,
	})
}

func (ev *eventHandler) handleDeploySubtaskEvent(e *proto.DeploySubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_DeploySubtaskEvent{
			DeploySubtaskEvent: e,
		},
	})
}
