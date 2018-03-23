/*
Copyright 2018 Google LLC

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
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

const quietPeriod = 500 * time.Millisecond

// WatcherFactory can build Watchers from a list of artifacts to be watched for changes
type WatcherFactory func(paths []string) (Watcher, error)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Start watches a set of artifacts for changes, and on the first change
	// returns a reference to the changed artifact
	Start(ctx context.Context, onChange func([]string))
}

// fsWatcher uses inotify to watch for changes and implements
// the Watcher interface
type fsWatcher struct {
	fsEvents chan notify.EventInfo
}

// NewWatcher creates a new Watcher on a list of artifacts.
func NewWatcher(paths []string) (Watcher, error) {
	// TODO(@dgageot): If file changes happen too quickly, events might be lost
	fsEvents := make(chan notify.EventInfo, 100)
	sort.Strings(paths)

	for _, p := range paths {
		logrus.Infof("Added watch for %s", p)
		if err := notify.Watch(p, fsEvents, notify.All); err != nil {
			notify.Stop(fsEvents)
			return nil, errors.Wrapf(err, "adding watch for %s", p)
		}
	}

	logrus.Info("Watch is ready")
	return &fsWatcher{
		fsEvents: fsEvents,
	}, nil
}

// Start watches a set of artifacts for changes with inotify, and on the first change
// returns a reference to the changed artifact
func (f *fsWatcher) Start(ctx context.Context, onChange func([]string)) {
	var changedPaths []string

	timer := time.NewTimer(1<<63 - 1) // Forever
	defer timer.Stop()

	for {
		select {
		case ei := <-f.fsEvents:
			logrus.Infof("%s %s", ei.Event().String(), ei.Path())
			changedPaths = append(changedPaths, ei.Path())
			timer.Reset(quietPeriod)
		case <-timer.C:
			onChange(changedPaths)
			changedPaths = nil
		case <-ctx.Done():
			notify.Stop(f.fsEvents)
			return
		}
	}
}
