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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

type contextKey struct{}

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
