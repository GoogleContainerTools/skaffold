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

package trigger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Trigger describes a mechanism that triggers the watch.
type Trigger interface {
	Start(context.Context) (<-chan bool, error)
	LogWatchToUser(io.Writer)
	Debounce() bool
}

type Config interface {
	Pipeline() latest.Pipeline
	Trigger() string
	WatchPollInterval() int
}

// NewTrigger creates a new trigger.
func NewTrigger(cfg Config, isActive func() bool) (Trigger, error) {
	switch strings.ToLower(cfg.Trigger()) {
	case "polling":
		return &pollTrigger{
			Interval: time.Duration(cfg.WatchPollInterval()) * time.Millisecond,
			isActive: isActive,
		}, nil
	case "notify":
		return newFSNotifyTrigger(cfg, isActive), nil
	case "manual":
		return &manualTrigger{
			isActive: isActive,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported trigger: %s", cfg.Trigger())
	}
}

func newFSNotifyTrigger(cfg Config, isActive func() bool) *fsNotifyTrigger {
	workspaces := map[string]struct{}{}
	for _, a := range cfg.Pipeline().Build.Artifacts {
		workspaces[a.Workspace] = struct{}{}
	}
	return &fsNotifyTrigger{
		Interval:   time.Duration(cfg.WatchPollInterval()) * time.Millisecond,
		workspaces: workspaces,
		isActive:   isActive,
		watchFunc:  notify.Watch,
	}
}

// pollTrigger watches for changes on a given interval of time.
type pollTrigger struct {
	Interval time.Duration
	isActive func() bool
}

// Debounce tells the watcher to debounce rapid sequence of changes.
func (t *pollTrigger) Debounce() bool {
	return true
}

func (t *pollTrigger) LogWatchToUser(out io.Writer) {
	if t.isActive() {
		color.Yellow.Fprintf(out, "Watching for changes every %v...\n", t.Interval)
	} else {
		color.Yellow.Fprintln(out, "Not watching for changes...")
	}
}

// Start starts a timer.
func (t *pollTrigger) Start(ctx context.Context) (<-chan bool, error) {
	trigger := make(chan bool)

	ticker := time.NewTicker(t.Interval)
	go func() {
		for {
			select {
			case <-ticker.C:

				// Ignore if trigger is inactive
				if !t.isActive() {
					continue
				}
				trigger <- true
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return trigger, nil
}

// manualTrigger watches for changes when the user presses a key.
type manualTrigger struct {
	isActive func() bool
}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *manualTrigger) Debounce() bool {
	return false
}

func (t *manualTrigger) LogWatchToUser(out io.Writer) {
	if t.isActive() {
		color.Yellow.Fprintln(out, "Press any key to rebuild/redeploy the changes")
	} else {
		color.Yellow.Fprintln(out, "Not watching for changes...")
	}
}

// Start starts listening to pressed keys.
func (t *manualTrigger) Start(ctx context.Context) (<-chan bool, error) {
	trigger := make(chan bool)

	var stopped int32
	go func() {
		<-ctx.Done()
		atomic.StoreInt32(&stopped, 1)
	}()

	reader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			_, _, err := reader.ReadRune()
			if err != nil {
				logrus.Debugf("manual trigger error: %s", err)
			}

			// Wait until the context is cancelled.
			if atomic.LoadInt32(&stopped) == 1 {
				return
			}

			// Ignore if trigger is inactive
			if !t.isActive() {
				continue
			}
			trigger <- true
		}
	}()

	return trigger, nil
}

// notifyTrigger watches for changes with fsnotify
type fsNotifyTrigger struct {
	Interval   time.Duration
	workspaces map[string]struct{}
	isActive   func() bool
	watchFunc  func(path string, c chan<- notify.EventInfo, events ...notify.Event) error
}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *fsNotifyTrigger) Debounce() bool {
	// This trigger has built-in debouncing.
	return false
}

func (t *fsNotifyTrigger) LogWatchToUser(out io.Writer) {
	if t.isActive() {
		color.Yellow.Fprintln(out, "Watching for changes...")
	} else {
		color.Yellow.Fprintln(out, "Not watching for changes...")
	}
}

// Start listening for file system changes
func (t *fsNotifyTrigger) Start(ctx context.Context) (<-chan bool, error) {
	c := make(chan notify.EventInfo, 100)

	// Workaround https://github.com/rjeczalik/notify/issues/96
	wd, err := RealWorkDir()
	if err != nil {
		return nil, err
	}

	// Watch current directory recursively
	if err := t.watchFunc(filepath.Join(wd, "..."), c, notify.All); err != nil {
		return nil, err
	}

	// Watch all workspaces recursively
	for w := range t.workspaces {
		if w == "." {
			continue
		}

		if err := t.watchFunc(filepath.Join(wd, w, "..."), c, notify.All); err != nil {
			return nil, err
		}
	}

	// Since the file watcher runs in a separate go routine
	// and can take some time to start, it can lose the very first change.
	// As a mitigation, we act as if a change was detected.
	go func() { c <- nil }()

	trigger := make(chan bool)
	go func() {
		timer := time.NewTimer(1<<63 - 1) // Forever

		for {
			select {
			case e := <-c:

				// Ignore detected changes if not active
				if !t.isActive() {
					continue
				}
				logrus.Debugln("Change detected", e)

				// Wait t.interval before triggering.
				// This way, rapid stream of events will be grouped.
				timer.Reset(t.Interval)
			case <-timer.C:
				trigger <- true
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()

	return trigger, nil
}

// StartTrigger attempts to start a trigger.
// It will attempt to start as a polling trigger if it tried unsuccessfully to start a notify trigger.
func StartTrigger(ctx context.Context, t Trigger) (<-chan bool, error) {
	ret, err := t.Start(ctx)
	if err == nil {
		return ret, err
	}
	if notifyTrigger, ok := t.(*fsNotifyTrigger); ok {
		logrus.Debugln("Couldn't start notify trigger. Falling back to a polling trigger")

		t = &pollTrigger{
			Interval: notifyTrigger.Interval,
			isActive: notifyTrigger.isActive,
		}
		ret, err = t.Start(ctx)
	}

	return ret, err
}
