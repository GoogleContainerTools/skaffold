package v2

import (
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
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

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			handler.handleDeploySubtaskEvent(test.event)
		})
	}
}
