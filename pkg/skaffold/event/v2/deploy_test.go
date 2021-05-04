package v2

import (
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestHandleDeploySubtaskEvent(t *testing.T) {
	tests := []struct{
		name  string
		event *proto.DeploySubtaskEvent
	}{
		{
			name: "In Progress",
			event: &proto.DeploySubtaskEvent{
				Id:       "0",
				TaskId:   fmt.Sprintf("%s-%d", constants.Deploy, 0),
				Status:   InProgress,
			},
		},
		{
			name: "Failed",
			event: &proto.DeploySubtaskEvent{
				Id:            "23",
				TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, 0),
				Status:        Failed,
				ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Deploy, errors.New("deploy failed")),
			},
		},
		{
			name: "Succeeded",
			event: &proto.DeploySubtaskEvent{
				Id:       "99",
				TaskId:   fmt.Sprintf("%s-%d", constants.Deploy, 12),
				Status:   Succeeded,
			},
		},
	}

	defer func() { handler = newHandler() }()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler = newHandler()
			handler.state = emptyState(mockCfg([]latest_v1.Pipeline{{}}, "test"))


			wait(t, func() bool { return handler.getState().DeployState.Status == NotStarted })
			handler.handleDeploySubtaskEvent(test.event)
			wait(t, func() bool { return handler.getState().DeployState.Status == test.event.Status })
		})
	}
}
