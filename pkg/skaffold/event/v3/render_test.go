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

func TestHandleRenderSubtaskEvent(t *testing.T) {

	renderStartedEvent := &proto.RenderStartedEvent{
		Id:     strconv.Itoa(1),
		TaskId: fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status: InProgress,
	}

	renderFailedEvent := &proto.RenderFailedEvent{
		Id:            strconv.Itoa(1),
		TaskId:        fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV3(handler.cfg, constants.Render, errors.New("render failed")),
	}

	renderSucceededEvent := &proto.RenderSucceededEvent{
		Id:     strconv.Itoa(1),
		TaskId: fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status: Succeeded,
	}

	t.Run("In Progress", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

		wait(t, func() bool { return handler.getState().RenderState.Status == NotStarted })
		handler.handle(renderStartedEvent.Id, renderStartedEvent, RenderStartedEvent)
		wait(t, func() bool { return handler.getState().RenderState.Status == InProgress })
	})

	t.Run("Failed", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

		wait(t, func() bool { return handler.getState().RenderState.Status == NotStarted })
		handler.handle(renderFailedEvent.Id, renderFailedEvent, RenderFailedEvent)
		wait(t, func() bool { return handler.getState().RenderState.Status == Failed })
	})

	t.Run("Succeeded", func(t *testing.T) {
		handler = newHandler()
		handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

		wait(t, func() bool { return handler.getState().RenderState.Status == NotStarted })
		handler.handle(renderSucceededEvent.Id, renderSucceededEvent, RenderSucceededEvent)
		wait(t, func() bool { return handler.getState().RenderState.Status == Succeeded })
	})
}
