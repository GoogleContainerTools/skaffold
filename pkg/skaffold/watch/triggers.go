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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/sirupsen/logrus"
)

// Trigger describes a mechanism that triggers the watch.
type Trigger interface {
	Start() (<-chan bool, func())
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
func (t *pollTrigger) Start() (<-chan bool, func()) {
	trigger := make(chan bool)

	ticker := time.NewTicker(t.Interval)
	go func() {
		for {
			<-ticker.C
			trigger <- true
		}
	}()

	return trigger, ticker.Stop
}

// manualTrigger watches for changes when the user presses a key.
type manualTrigger struct {
}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *manualTrigger) Debounce() bool {
	return false
}

func (t *manualTrigger) WatchForChanges(out io.Writer) {
	color.Yellow.Fprintln(out, "Press any key to rebuild/redeploy the changes")
}

// Start starts listening to pressed keys.
func (t *manualTrigger) Start() (<-chan bool, func()) {
	trigger := make(chan bool)

	reader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			_, _, err := reader.ReadRune()
			if err != nil {
				logrus.Debugf("manual trigger error: %s", err)
			}
			trigger <- true
		}
	}()

	return trigger, func() {}
}
