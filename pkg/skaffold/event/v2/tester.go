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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TesterInProgress(id int) {
	handler.handleTestSubtaskEvent(&proto.TestSubtaskEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: InProgress,
	})
}

func TesterFailed(id int, err error) {
	handler.handleTestSubtaskEvent(&proto.TestSubtaskEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Test, err),
	})
}

func TesterSucceeded(id int) {
	handler.handleTestSubtaskEvent(&proto.TestSubtaskEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: Succeeded,
	})
}

func (ev *eventHandler) handleTestSubtaskEvent(e *proto.TestSubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_TestEvent{
			TestEvent: e,
		},
	})
}
