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
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto/enums"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

type logger struct {
	Phase     constants.Phase
	SubtaskID string
}

func NewLogger(phase constants.Phase, subtaskID string) io.Writer {
	return logger{
		Phase:     phase,
		SubtaskID: subtaskID,
	}
}

func (l logger) Write(p []byte) (int, error) {
	handler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", l.Phase, handler.iteration),
		SubtaskId: l.SubtaskID,
		Level:     -1,
		Message:   string(p),
	})

	return len(p), nil
}

func (ev *eventHandler) handleSkaffoldLogEvent(e *proto.SkaffoldLogEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_SkaffoldLogEvent{
			SkaffoldLogEvent: e,
		},
	})
}

// logHook is an implementation of logrus.Hook used to send SkaffoldLogEvents
type logHook struct{}

func NewLogHook() logrus.Hook {
	return logHook{}
}

// Levels returns all levels as we want to send events for all levels
func (h logHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

// Fire constructs a SkaffoldLogEvent and sends it to the event channel
func (h logHook) Fire(entry *logrus.Entry) error {
	handler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", handler.task, handler.iteration),
		SubtaskId: SubtaskIDNone,
		Level:     levelFromEntry(entry),
		Message:   entry.Message,
	})
	return nil
}

func levelFromEntry(entry *logrus.Entry) enums.LogLevel {
	switch entry.Level {
	case logrus.FatalLevel:
		return enums.LogLevel_FATAL
	case logrus.ErrorLevel:
		return enums.LogLevel_ERROR
	case logrus.WarnLevel:
		return enums.LogLevel_WARN
	case logrus.InfoLevel:
		return enums.LogLevel_INFO
	case logrus.PanicLevel:
		return enums.LogLevel_PANIC
	case logrus.TraceLevel:
		return enums.LogLevel_TRACE
	default:
		return enums.LogLevel_DEBUG
	}
}
