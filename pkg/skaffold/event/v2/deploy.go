package v2

import (
	"fmt"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func DeployInProgress(id int) {
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:       strconv.Itoa(id),
		TaskId:   fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status:   InProgress,
	})
}

func DeployFailed(id int, err error) {
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Deploy, err),
	})
}

func DeploySucceeded(id int) {
	handler.handleDeploySubtaskEvent(&proto.DeploySubtaskEvent{
		Id:       strconv.Itoa(id),
		TaskId:   fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Status:   Succeeded,
	})
}

func (ev *eventHandler) handleDeploySubtaskEvent(e *proto.DeploySubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_DeploySubtaskEvent{
			DeploySubtaskEvent: e,
		},
	})
}
