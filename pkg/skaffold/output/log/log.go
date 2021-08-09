package log

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/sirupsen/logrus"
)

// Entry takes an context.Context and constructs a logrus.Entry from it, adding
// fields for task and subtask information
func Entry(ctx context.Context) *logrus.Entry {
	task := ctx.Value("task")
	subtask := ctx.Value("subtask")
	if task != nil && subtask != nil {
		task := task.(string)
		subtask := subtask.(string)
		return logrus.WithFields(logrus.Fields{
			"task":    task,
			"subtask": subtask,
		})
	}

	// Use constants.DevLoop as the default task, as it's the highest level task we
	// can default to if one isn't specified.
	return logrus.WithFields(logrus.Fields{
		"task":    constants.DevLoop,
		"subtask": eventV2.SubtaskIDNone,
	})
}
