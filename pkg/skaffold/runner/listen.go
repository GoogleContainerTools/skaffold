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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
)

type Listener interface {
	WatchForChanges(context.Context, io.Writer, func() needs, func(context.Context, io.Writer, needs) error) error
	LogWatchToUser(io.Writer)
}

type needs struct {
	needsSync   bool
	needsBuild  bool
	needsDeploy bool
	work        devWorkItems
}

func (n *needs) needsNewLoop() bool {
	return n.needsSync || n.needsBuild || n.needsDeploy
}

func (n *needs) Clone() needs {
	return needs{
		needsSync:   n.needsSync,
		needsBuild:  n.needsBuild,
		needsDeploy: n.needsDeploy,
		work:        n.work.Clone(),
	}
}

type SkaffoldListener struct {
	Monitor    filemon.Monitor
	Trigger    trigger.Trigger
	intentChan <-chan bool
	ctxDev     context.Context
	cancelDev  context.CancelFunc
}

func (l *SkaffoldListener) LogWatchToUser(out io.Writer) {
	l.Trigger.LogWatchToUser(out)
}

// WatchForChanges listens to a trigger, and when one is received, computes file changes and
// conditionally runs the dev loop.
func (l *SkaffoldListener) WatchForChanges(ctx context.Context, out io.Writer, devChecker func() needs, devLoop func(context.Context, io.Writer, needs) error) error {
	ctxTrigger, cancelTrigger := context.WithCancel(ctx)
	defer cancelTrigger()
	trigger, err := trigger.StartTrigger(ctxTrigger, l.Trigger)
	if err != nil {
		return fmt.Errorf("unable to start trigger: %w", err)
	}

	// exit if file monitor fails the first time
	if err := l.Monitor.Run(l.Trigger.Debounce()); err != nil {
		return fmt.Errorf("failed to monitor files: %w", err)
	}

	l.LogWatchToUser(out)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-l.intentChan:
			if err := l.startDevInBackground(ctx, out, devChecker, devLoop); err != nil {
				return err
			}
		case <-trigger:
			if err := l.startDevInBackground(ctx, out, devChecker, devLoop); err != nil {
				return err
			}
		}
	}
}

func (l *SkaffoldListener) startDevInBackground(ctx context.Context, out io.Writer, checker func() needs, devLoop func(context.Context, io.Writer, needs) error) error {
	if err := l.Monitor.Run(l.Trigger.Debounce()); err != nil {
		logrus.Warnf("Ignoring changes: %s", err.Error())
		return nil
	}
	n := checker()
	if n.work.needsReload {
		return ErrorConfigurationChanged
	}
	if !n.needsNewLoop() {
		return nil
	}
	l.Monitor.Reset()
	if l.cancelDev != nil {
		l.cancelDev()
	}
	l.ctxDev, l.cancelDev = context.WithCancel(ctx)
	go func(ctx context.Context) {
		if err := devLoop(ctx, out, n); err != nil {
			logrus.Errorf("error running dev loop: %s", err.Error())
		}
	}(l.ctxDev)
	return nil
}
