/*
Copyright 2019 The Skaffold Authors

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

package cluster

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLogLevel(t *testing.T) {
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
		defer func(l logrus.Level) { logrus.SetLevel(l) }(logrus.GetLevel())
		logrus.SetLevel(test.logrusLevel)

		kanikoLevel := logLevel()

		testutil.CheckDeepEqual(t, test.expected, kanikoLevel)
	}
}
