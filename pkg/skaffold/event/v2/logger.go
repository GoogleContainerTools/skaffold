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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

type logger struct {
	Phase     constants.Phase
	SubtaskID string
	Origin    string
}

func NewLogger(phase constants.Phase, subtaskID, origin string) io.Writer {
	return logger{
		Phase:     phase,
		SubtaskID: subtaskID,
		Origin:    origin,
	}
}

func (l logger) Write(p []byte) (int, error) {
	handler.handleSkaffoldLogEvent(&proto.SkaffoldLogEvent{
		TaskId:    fmt.Sprintf("%s-%d", l.Phase, handler.iteration),
		SubtaskId: l.SubtaskID,
		Origin:    l.Origin,
		Level:     0,
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
