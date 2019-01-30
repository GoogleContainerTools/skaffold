/*
Copyright 2018 The Skaffold Authors

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

package watch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

// Trigger describes a mechanism that triggers the watch.
type Trigger interface {
	Start(context.Context) (<-chan bool, error)
	WatchForChanges(io.Writer)
	Debounce() bool
}

// NewTrigger creates a new trigger.
func NewTrigger(opts *config.SkaffoldOptions) (Trigger, error) {
	switch strings.ToLower(opts.Trigger) {
	case "polling":
		return &pollTrigger{
			Interval: time.Duration(opts.WatchPollInterval) * time.Millisecond,
		}, nil
	case "notify":
		return &fsNotifyTrigger{
			Interval: time.Duration(opts.WatchPollInterval) * time.Millisecond,
		}, nil
	case "manual":
		return &manualTrigger{}, nil
	default:
		return nil, fmt.Errorf("unsupported type of trigger: %s", opts.Trigger)
	}
}

// pollTrigger watches for changes on a given interval of time.
type pollTrigger struct {
	Interval time.Duration
}

// Debounce tells the watcher to debounce rapid sequence of changes.
func (t *pollTrigger) Debounce() bool {
	return true
}

func (t *pollTrigger) WatchForChanges(out io.Writer) {
	color.Yellow.Fprintf(out, "Watching for changes every %v...\n", t.Interval)
}

// Start starts a timer.
func (t *pollTrigger) Start(ctx context.Context) (<-chan bool, error) {
	trigger := make(chan bool)

	ticker := time.NewTicker(t.Interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				trigger <- true
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()

	return trigger, nil
}

// manualTrigger watches for changes when the user presses a key.
type manualTrigger struct{}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *manualTrigger) Debounce() bool {
	return false
}

func (t *manualTrigger) WatchForChanges(out io.Writer) {
	color.Yellow.Fprintln(out, "Press any key to rebuild/redeploy the changes")
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
			trigger <- true
		}
	}()

	return trigger, nil
}

// notifyTrigger watches for changes with fsnotify
type fsNotifyTrigger struct {
	Interval time.Duration
}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *fsNotifyTrigger) Debounce() bool {
	// This trigger has built-in debouncing.
	return false
}

func (t *fsNotifyTrigger) WatchForChanges(out io.Writer) {
	color.Yellow.Fprintln(out, "Watching for changes...")
}

// Start Listening for file system changes
func (t *fsNotifyTrigger) Start(ctx context.Context) (<-chan bool, error) {
	// TODO(@dgageot): If file changes happen too quickly, events might be lost
	c := make(chan notify.EventInfo, 100)

	// Watch current directory recursively
	if err := notify.Watch("./...", c, notify.All); err != nil {
		return nil, err
	}

	trigger := make(chan bool)
	go func() {
		timer := time.NewTimer(1<<63 - 1) // Forever

		for {
			select {
			case e := <-c:
				logrus.Debugln("Change detected", e)

				// Wait t.interval before triggering.
				// This way, rapid stream of events will be grouped.
				timer.Reset(t.Interval)
			case <-timer.C:
				trigger <- true
			case <-ctx.Done():
				timer.Stop()
			}
		}
	}()

	return trigger, nil
}
