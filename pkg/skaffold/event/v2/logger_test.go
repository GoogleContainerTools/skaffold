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
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/proto/enums"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestHandleSkaffoldLogEvent(t *testing.T) {
	testHandler := newHandler()
	testHandler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

	messages := []string{
		"hi!",
		"how's it going",
		"hope you're well",
		"this is a skaffold test",
	}

	// ensure that messages sent through the SkaffoldLog function are populating the event log
	for _, message := range messages {
		testHandler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
			TaskId:    "Test-0",
			SubtaskId: "1",
			Level:     enums.LogLevel_INFO,
			Message:   message,
		})
	}
	wait(t, func() bool {
		testHandler.skaffoldLogsLock.Lock()
		logLen := len(testHandler.skaffoldLogs)
		testHandler.skaffoldLogsLock.Unlock()
		return logLen == len(messages)
	})
}
