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
    "errors"
    "fmt"
    "io"

    "github.com/sirupsen/logrus"

    "github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
    "github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
)

type Listener interface {
    WatchForChanges(context.Context, io.Writer, func(context.Context, context.Context, io.Writer, bool) error) error
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
func (l *SkaffoldListener) WatchForChanges(ctx context.Context, out io.Writer, devLoop func(context.Context, context.Context, io.Writer, bool) error) error {

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

    // Start the devloop trigger in another go routine.
    errCh := make(chan error, 1)
    isFirst := true
    var prevCancel context.CancelFunc
    for {
        l.LogWatchToUser(out)
        sCtx, cancel := context.WithCancel(ctx)
        defer cancel()
        select {
        case <-ctx.Done():
            fmt.Fprintln(out, "done- for")
            errCh <- nil
        case <-l.intentChan:
            if !isFirst{prevCancel()}
            go func() {
                if err := l.do(ctx, sCtx, out, isFirst, devLoop); err != nil {
                    errCh <- err
                }
            }()
        case <-trigger:
            if !isFirst{prevCancel()}
            go func() {
                if err := l.do(ctx, sCtx, out, isFirst, devLoop); err != nil {
                    errCh <- err
                }
            }()
        }
        isFirst = false
        prevCancel = cancel
    }
    fmt.Fprintln(out, "done")
	// Wait for either any error in the dev loop or user hit cancels.
    e := <-errCh
    if isFirst {
		// First Dev loop
    	return fmt.Errorf("exiting dev mode because first deploy failed: %w", e)
	}
    return e
}

func (l *SkaffoldListener) do(ctx context.Context, sCtx context.Context, out io.Writer, isFirst bool, devLoop func(context.Context, context.Context, io.Writer, bool) error) error {
    if !isFirst {
        if err := l.Monitor.Run(l.Trigger.Debounce()); err != nil {
            logrus.Warnf("error computing file changes: %s", err.Error())
            logrus.Warnf("skaffold may not run successfully!")
        }
    }
    if err := devLoop(ctx, sCtx, out, isFirst); err != nil {
        // propagating this error up causes a new runner to be created
        // and a new dev loop to start
        if errors.Is(err, ErrorConfigurationChanged) {
            return err
        }
        if isFirst {
            return err
        }
        logrus.Errorf("error running dev loop: %s", err.Error())
    }
    return nil
}
