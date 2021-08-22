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
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

func TestHandleApplicationLogEvent(t *testing.T) {
	testHandler := newHandler()
	testHandler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

	messages := []string{
		"hi!",
		"how's it going",
		"hope you're well",
	}

	// ensure that messages sent through the ApplicationLog function are populating the event log
	for _, message := range messages {
		testHandler.handleApplicationLogEvent(&proto.ApplicationLogEvent{
			ContainerName: "containerName-0",
			PodName:       "pod-0",
			Message:       message,
		})
	}
	wait(t, func() bool {
		testHandler.applicationLogsLock.Lock()
		logLen := len(testHandler.applicationLogs)
		testHandler.applicationLogsLock.Unlock()
		return logLen == len(messages)
	})
}
