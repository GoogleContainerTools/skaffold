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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestHandleTestSubtaskEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *proto.TestSubtaskEvent
	}{
		{
			name: "In Progress",
			event: &proto.TestSubtaskEvent{
				Id:     "0",
				TaskId: fmt.Sprintf("%s-%d", constants.Test, 0),
				Status: InProgress,
			},
		},
		{
			name: "Failed",
			event: &proto.TestSubtaskEvent{
				Id:            "23",
				TaskId:        fmt.Sprintf("%s-%d", constants.Test, 0),
				Status:        Failed,
				ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Test, errors.New("deploy failed")),
			},
		},
		{
			name: "Succeeded",
			event: &proto.TestSubtaskEvent{
				Id:     "99",
				TaskId: fmt.Sprintf("%s-%d", constants.Test, 12),
				Status: Succeeded,
			},
		},
	}

	defer func() { handler = newHandler() }()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler = newHandler()
			handler.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

			wait(t, func() bool { return handler.getState().TestState.Status == NotStarted })
			handler.handleTestSubtaskEvent(test.event)
			wait(t, func() bool { return handler.getState().TestState.Status == test.event.Status })
		})
	}
}
