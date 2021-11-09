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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestTaskFailed(t *testing.T) {
	tcs := []struct {
		description string
		state       *proto.State
		phase       constants.Phase
		waitFn      func() bool
	}{
		{
			description: "build failed",
			phase:       constants.Build,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "Build-0"
			},
		},
		{
			description: "deploy failed",
			phase:       constants.Deploy,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "Deploy-0"
			},
		},
		{
			description: "status check failed",
			phase:       constants.StatusCheck,
			waitFn: func() bool {
				handler.logLock.Lock()
				logEntry := handler.eventLog[len(handler.eventLog)-1]
				handler.logLock.Unlock()
				te := logEntry.GetTaskEvent()
				return te != nil && te.Status == Failed && te.Id == "StatusCheck-0"
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			TaskFailed(tc.phase, errors.New("random error"))
			wait(t, tc.waitFn)
		})
	}
}
