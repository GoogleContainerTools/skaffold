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
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	protoV3 "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func TesterInProgress(id int) {
	event := &protoV3.TestStartedEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: InProgress,
	}
	updateTestStatus(event.Status)
	handler.handle(event.TaskId, event, TestStartedEvent)
}

func TesterFailed(id int, err error) {
	event := &protoV3.TestFailedEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV3(handler.cfg, constants.Test, err),
	}
	updateTestStatus(event.Status)
	handler.handle(event.TaskId, event, TestFailedEvent)
}

func TesterSucceeded(id int) {
	event := &protoV3.TestSucceededEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: Succeeded,
	}
	updateTestStatus(event.Status)
	handler.handle(event.TaskId, event, TestSucceededEvent)
}

func updateTestStatus(status string) {
	handler.stateLock.Lock()
	handler.state.TestState.Status = status
	handler.stateLock.Unlock()
}
