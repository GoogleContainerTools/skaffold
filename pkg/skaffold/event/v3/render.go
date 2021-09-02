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

// RendererInProgress adds an event to mark a render process starts.
func RendererInProgress(id int) {
	rendererEvent := &protoV3.RenderStartedEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status: InProgress,
	}
	updateRenderStatus(rendererEvent.Status)
	handler.handle(rendererEvent, RenderStartedEvent)
}

// RendererFailed adds an event to mark a render process has been failed.
func RendererFailed(id int, err error) {
	rendererEvent := &protoV3.RenderFailedEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV3(handler.cfg, constants.Render, err),
	}
	updateRenderStatus(rendererEvent.Status)
	handler.handle(rendererEvent, RenderFailedEvent)
}

// RendererSucceeded adds an event to mark a render process has been succeeded.
func RendererSucceeded(id int) {
	rendererEvent := &protoV3.RenderSucceededEvent{
		Id:     strconv.Itoa(id),
		TaskId: fmt.Sprintf("%s-%d", constants.Render, handler.iteration),
		Status: Succeeded,
	}
	updateRenderStatus(rendererEvent.Status)
	handler.handle(rendererEvent, RenderSucceededEvent)
}

func updateRenderStatus(status string) {
	handler.stateLock.Lock()
	handler.state.RenderState.Status = status
	handler.stateLock.Unlock()
}
