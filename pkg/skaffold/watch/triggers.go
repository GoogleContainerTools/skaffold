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
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

// Trigger describes a mechanism that triggers the watch.
type Trigger interface {
	Start(io.Writer, []*component) (<-chan bool, func(), error)
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
		return &fsNotifyTrigger{}, nil
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
func (t *pollTrigger) Start(out io.Writer, components []*component) (<-chan bool, func(), error) {
	trigger := make(chan bool)

	ticker := time.NewTicker(t.Interval)
	go func() {
		for {
			<-ticker.C
			trigger <- true
		}
	}()

	return trigger, ticker.Stop, nil
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
func (t *manualTrigger) Start(out io.Writer, components []*component) (<-chan bool, func(), error) {
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

	return trigger, func() {}, nil
}

// notifyTrigger watches for changes when fsnotify
type fsNotifyTrigger struct {
}

// Debounce tells the watcher to not debounce rapid sequence of changes.
func (t *fsNotifyTrigger) Debounce() bool {
	return false
}

func (t *fsNotifyTrigger) WatchForChanges(out io.Writer) {
	color.Yellow.Fprintln(out, "Watching for changes on directory using notifications")
}

// Start Listening for file system changes
func (t *fsNotifyTrigger) Start(out io.Writer, components []*component) (<-chan bool, func(), error) {
	trigger := make(chan bool)
	c := make(chan notify.EventInfo, 1)

	basePath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	if err := notify.Watch("./...", c, notify.All); err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			ei := <-c
			if isFileModifiedBelongsToComponent(basePath, ei.Path(), components) {
				color.Yellow.Fprintln(out, "Triggering rebuild because of ", cleanupFile(basePath, ei.Path()))
				trigger <- true
			}
		}
	}()
	return trigger, func() {
		notify.Stop(c)
	}, nil
}

func cleanupFile(basePath string, path string) string {
	file := filepath.Clean(path)
	file, _ = filepath.Abs(file)
	file = strings.Replace(file, basePath, "", 1)
	file = strings.Replace(file, "___jb_tmp___", "", 1)
	return file
}

func isFileModifiedBelongsToComponent(basePath string, path string, components []*component) bool {
	file := cleanupFile(basePath, path)
	logrus.Debugf("Testing if file :%s is part of a component", file)
	for _, component := range components {
		_, ok := component.state[file]
		if ok {
			return true
		}
	}
	logrus.Debugf("File %s is not part of the components", file)
	return false

}
