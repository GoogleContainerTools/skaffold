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

package runner

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Listener interface {
	WatchForChanges(context.Context, io.Writer, func(context.Context, io.Writer) error) error
	LogWatchToUser(io.Writer)
}

type SkaffoldListener struct {
	Monitor    filemon.Monitor
	Trigger    trigger.Trigger
	intentChan <-chan bool
}

func (l *SkaffoldListener) LogWatchToUser(out io.Writer) {
	l.Trigger.LogWatchToUser(out)
}

// WatchForChanges listens to a trigger, and when one is received, computes file changes and
// conditionally runs the dev loop.
func (l *SkaffoldListener) WatchForChanges(ctx context.Context, out io.Writer, devLoop func(context.Context, io.Writer) error) error {
	ctxTrigger, cancelTrigger := context.WithCancel(ctx)
	defer cancelTrigger()
	trigger, err := trigger.StartTrigger(ctxTrigger, l.Trigger)
	if err != nil {
		return errors.Wrap(err, "unable to start trigger")
	}

	// exit if file monitor fails the first time
	if err := l.Monitor.Run(l.Trigger.Debounce()); err != nil {
		return errors.Wrap(err, "failed to monitor files")
	}

	l.LogWatchToUser(out)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-l.intentChan:
			if err := l.do(ctx, out, devLoop); err != nil {
				return err
			}
		case <-trigger:
			if err := l.do(ctx, out, devLoop); err != nil {
				return err
			}
		}
	}
}

func (l *SkaffoldListener) do(ctx context.Context, out io.Writer, devLoop func(context.Context, io.Writer) error) error {
	if err := l.Monitor.Run(l.Trigger.Debounce()); err != nil {
		logrus.Warnf("error computing file changes: %s", err.Error())
		logrus.Warnf("skaffold may not run successfully!")
	}
	if err := devLoop(ctx, out); err != nil {
		// propagating this error up causes a new runner to be created
		// and a new dev loop to start
		if errors.Cause(err) == ErrorConfigurationChanged {
			return err
		}
		logrus.Errorf("error running dev loop: %s", err.Error())
	}
	return nil
}
