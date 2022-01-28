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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

// VerifyInProgress adds an event to mark a render process starts.
func VerifyInProgress(name string) {
	handler.handleVerifySubtaskEvent(&proto.VerifySubtaskEvent{
		Id:     name,
		TaskId: fmt.Sprintf("%s-%d", constants.Verify, handler.iteration),
		Status: InProgress,
	})
}

// VerifyFailed adds an event to mark a render process has been failed.
func VerifyFailed(name string, err error) {
	handler.handleVerifySubtaskEvent(&proto.VerifySubtaskEvent{
		Id:            name,
		TaskId:        fmt.Sprintf("%s-%d", constants.Verify, handler.iteration),
		Status:        Failed,
		ActionableErr: sErrors.ActionableErrV2(handler.cfg, constants.Verify, err),
	})
}

// VerifySucceeded adds an event to mark a render process has been succeeded.
func VerifySucceeded(name string) {
	handler.handleVerifySubtaskEvent(&proto.VerifySubtaskEvent{
		Id:     name,
		TaskId: fmt.Sprintf("%s-%d", constants.Verify, handler.iteration),
		Status: Succeeded,
	})
}

func (ev *eventHandler) handleVerifySubtaskEvent(e *proto.VerifySubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_VerifyEvent{
			VerifyEvent: e,
		},
	})
}
