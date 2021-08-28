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
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func TestHandleTestSubtaskEvent(t *testing.T) {

	testFailedEvent := &proto.TestFailedEvent{
		Id:            strconv.Itoa(1),
		TaskId:        fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV3(handler.cfg, constants.Deploy, errors.New("status check failed")),
	}

	testStartedEvent := &proto.TestStartedEvent{
		Id:     strconv.Itoa(23),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: InProgress,
	}

	testSucceededEvent := &proto.TestSucceededEvent{
		Id:     strconv.Itoa(99),
		TaskId: fmt.Sprintf("%s-%d", constants.Test, handler.iteration),
		Status: Succeeded,
	}

	t.Run("In Progress", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

		wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
		handler.handle(testStartedEvent.Id, testStartedEvent, TestStartedEvent)
		wait(t, func() bool { return handler.getState().TestState.Status == InProgress })
	})

	t.Run("Failed", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

		wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
		handler.handle(testFailedEvent.Id, testFailedEvent, TestFailedEvent)
		wait(t, func() bool { return handler.getState().TestState.Status == Failed })
	})

	t.Run("Succeeded", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))
		wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })

		handler.handle(testSucceededEvent.Id, testSucceededEvent, TestSucceededEvent)
		wait(t, func() bool { return handler.getState().TestState.Status == Succeeded })
	})

}
