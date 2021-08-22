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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func TestHandleDeploySubtaskEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *proto.DeploySubtaskEvent
	}{
		{
			name: "In Progress",
			event: &proto.DeploySubtaskEvent{
				Id:     "0",
				TaskId: fmt.Sprintf("%s-%d", constants.Deploy, 0),
				Status: InProgress,
			},
		},
		{
			name: "Failed",
			event: &proto.DeploySubtaskEvent{
				Id:            "23",
				TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, 0),
				Status:        Failed,
				ActionableErr: sErrors.ActionableErrV3(handler.cfg, constants.Deploy, errors.New("deploy failed")),
			},
		},
		{
			name: "Succeeded",
			event: &proto.DeploySubtaskEvent{
				Id:     "99",
				TaskId: fmt.Sprintf("%s-%d", constants.Deploy, 12),
				Status: Succeeded,
			},
		},
	}

	defer func() { handler = newHandler() }()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler = newHandler()
			handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

			wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
			handler.handleDeploySubtaskEvent(test.event)
			wait(t, func() bool { return handler.getState().DeployState.Status == test.event.Status })
		})
	}
}
