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

package log

import (
	"context"
	"testing"

	ggcrlogs "github.com/google/go-containerregistry/pkg/logs"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestEntry(t *testing.T) {
	tests := []struct {
		name            string
		task            constants.Phase
		expectedTask    constants.Phase
		subtask         string
		expectedSubtask string
		emptyContext    bool
	}{
		{
			name:            "arbitrary task and subtask values",
			task:            constants.Build,
			subtask:         "test",
			expectedTask:    constants.Build,
			expectedSubtask: "test",
		},
		{
			name:            "context missing values",
			emptyContext:    true,
			expectedTask:    constants.DevLoop,
			expectedSubtask: constants.SubtaskIDNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if !test.emptyContext {
				ctx = context.WithValue(ctx, ContextKey, EventContext{
					Task:    test.task,
					Subtask: test.subtask,
				})
			}

			got := Entry(ctx)
			testutil.CheckDeepEqual(t, test.expectedTask, got.Data["task"])
			testutil.CheckDeepEqual(t, test.expectedSubtask, got.Data["subtask"])
		})
	}
}

func TestKanikoLogLevel(t *testing.T) {
	tests := []struct {
		logrusLevel logrus.Level
		expected    logrus.Level
	}{
		{logrusLevel: logrus.TraceLevel, expected: logrus.DebugLevel},
		{logrusLevel: logrus.DebugLevel, expected: logrus.DebugLevel},
		{logrusLevel: logrus.InfoLevel, expected: logrus.InfoLevel},
		{logrusLevel: logrus.WarnLevel, expected: logrus.InfoLevel},
		{logrusLevel: logrus.ErrorLevel, expected: logrus.InfoLevel},
		{logrusLevel: logrus.FatalLevel, expected: logrus.InfoLevel},
		{logrusLevel: logrus.PanicLevel, expected: logrus.InfoLevel},
	}
	for _, test := range tests {
		defer func(l logrus.Level) { logger.SetLevel(l) }(logger.GetLevel())
		logger.SetLevel(test.logrusLevel)

		kanikoLevel := KanikoLogLevel()

		testutil.CheckDeepEqual(t, test.expected, kanikoLevel)
	}
}

func TestSetupGGCRLogging(t *testing.T) {
	tests := []struct {
		description           string
		logLevel              logrus.Level
		expectWarnEnabled     bool
		expectProgressEnabled bool
		expectDebugEnabled    bool
	}{
		{
			description: "fatal log level disables ggcr logging",
			logLevel:    logrus.FatalLevel,
		},
		{
			description:       "error log level enables ggcr warn logging",
			logLevel:          logrus.ErrorLevel,
			expectWarnEnabled: true,
		},
		{
			description:       "warn log level enables ggcr warn logging",
			logLevel:          logrus.WarnLevel,
			expectWarnEnabled: true,
		},
		{
			description:           "info log level enables ggcr warn and progress logging",
			logLevel:              logrus.InfoLevel,
			expectWarnEnabled:     true,
			expectProgressEnabled: true,
		},
		{
			description:           "debug log level enables ggcr warn and progress logging",
			logLevel:              logrus.DebugLevel,
			expectWarnEnabled:     true,
			expectProgressEnabled: true,
		},
		{
			description:           "trace log level enables ggcr warn and progress and debug logging",
			logLevel:              logrus.TraceLevel,
			expectWarnEnabled:     true,
			expectProgressEnabled: true,
			expectDebugEnabled:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			setupGGCRLogging(logrus.New(), test.logLevel)
			t.CheckDeepEqual(test.expectWarnEnabled, ggcrlogs.Enabled(ggcrlogs.Warn))
			t.CheckDeepEqual(test.expectProgressEnabled, ggcrlogs.Enabled(ggcrlogs.Progress))
			t.CheckDeepEqual(test.expectDebugEnabled, ggcrlogs.Enabled(ggcrlogs.Debug))
			if test.expectWarnEnabled {
				t.CheckDeepEqual(0, ggcrlogs.Warn.Flags())
			}
			if test.expectProgressEnabled {
				t.CheckDeepEqual(0, ggcrlogs.Progress.Flags())
			}
			if test.expectDebugEnabled {
				t.CheckDeepEqual(0, ggcrlogs.Debug.Flags())
			}
		})
	}
}
