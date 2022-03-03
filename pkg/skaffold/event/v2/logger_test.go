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

	"github.com/sirupsen/logrus"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/proto/enums"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
		testHandler.logLock.Lock()
		logLen := len(testHandler.eventLog)
		testHandler.logLock.Unlock()
		return logLen == len(messages)
	})
}

func TestLevelFromEntry(t *testing.T) {
	tests := []struct {
		name      string
		logrusLvl logrus.Level
		enumLvl   enums.LogLevel
	}{
		{
			name:      "panic",
			logrusLvl: logrus.PanicLevel,
			enumLvl:   enums.LogLevel_PANIC,
		},
		{
			name:      "fatal",
			logrusLvl: logrus.FatalLevel,
			enumLvl:   enums.LogLevel_FATAL,
		},
		{
			name:      "error",
			logrusLvl: logrus.ErrorLevel,
			enumLvl:   enums.LogLevel_ERROR,
		},
		{
			name:      "warn",
			logrusLvl: logrus.WarnLevel,
			enumLvl:   enums.LogLevel_WARN,
		},
		{
			name:      "info",
			logrusLvl: logrus.InfoLevel,
			enumLvl:   enums.LogLevel_INFO,
		},
		{
			name:      "debug",
			logrusLvl: logrus.DebugLevel,
			enumLvl:   enums.LogLevel_DEBUG,
		},
		{
			name:      "trace",
			logrusLvl: logrus.TraceLevel,
			enumLvl:   enums.LogLevel_TRACE,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			got := levelFromEntry(&logrus.Entry{Level: test.logrusLvl})
			t.CheckDeepEqual(test.enumLvl, got)
		})
	}
}
