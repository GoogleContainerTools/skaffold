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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// DefaultLogLevel is logrus warn level
const DefaultLogLevel = logrus.WarnLevel

type contextKey struct{}

var AllLevels = logrus.AllLevels
var ContextKey = contextKey{}

type EventContext struct {
	Task    constants.Phase
	Subtask string
}

// Entry takes an context.Context and constructs a logrus.Entry from it, adding
// fields for task and subtask information
func Entry(ctx context.Context) *logrus.Entry {
	val := ctx.Value(ContextKey)
	if eventContext, ok := val.(EventContext); ok {
		return logrus.WithFields(logrus.Fields{
			"task":    eventContext.Task,
			"subtask": eventContext.Subtask,
		})
	}

	// Use constants.DevLoop as the default task, as it's the highest level task we
	// can default to if one isn't specified.
	return logrus.WithFields(logrus.Fields{
		"task":    constants.DevLoop,
		"subtask": constants.SubtaskIDNone,
	})
}

// IsDebugLevelEnabled returns true if debug level log is enabled.
func IsDebugLevelEnabled() bool {
	return logrus.IsLevelEnabled(logrus.DebugLevel)
}

// IsTraceLevelEnabled returns true if trace level log is enabled.
func IsTraceLevelEnabled() bool {
	return logrus.IsLevelEnabled(logrus.TraceLevel)
}

func New() *logrus.Logger {
	return logrus.New()
}

// KanikoLogLevel makes sure kaniko logs at least at Info level and at most Debug level (trace doesn't work with Kaniko)
func KanikoLogLevel() logrus.Level {
	level := logrus.GetLevel()
	if level < logrus.InfoLevel {
		return logrus.InfoLevel
	}
	if level > logrus.DebugLevel {
		return logrus.DebugLevel
	}
	return level
}

// SetupLogs sets up logrus logger for skaffold command line
func SetupLogs(stdErr io.Writer, level string, timestamp bool, hook logrus.Hook) error {
	logrus.SetOutput(stdErr)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: timestamp,
	})
	logrus.AddHook(hook)
	return nil
}
