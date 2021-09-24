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
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestHandleRenderSubtaskEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *proto.RenderSubtaskEvent
	}{
		{
			name: "In Progress",
			event: &proto.RenderSubtaskEvent{
				Id:     "0",
				TaskId: fmt.Sprintf("%s-%d", constants.Render, 0),
				Status: InProgress,
			},
		},
		{
			name: "Failed",
			event: &proto.RenderSubtaskEvent{
				Id:            "23",
				TaskId:        fmt.Sprintf("%s-%d", constants.Render, 0),
				Status:        Failed,
				ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Render, errors.New("render failed")),
			},
		},
		{
			name: "Succeeded",
			event: &proto.RenderSubtaskEvent{
				Id:     "99",
				TaskId: fmt.Sprintf("%s-%d", constants.Render, 12),
				Status: Succeeded,
			},
		},
	}

	defer func() { handler = newHandler() }()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler = newHandler()
			handler.state = emptyState(mockCfg([]latestV2.Pipeline{{}}, "test"))
			wait(t, func() bool { return handler.getState().RenderState.Status == NotStarted })
			handler.handleRenderSubtaskEvent(test.event)
			wait(t, func() bool { return handler.getState().RenderState.Status == test.event.Status })
		})
	}
}
