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
	"path/filepath"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const quietPeriod = 500 * time.Millisecond

// WatcherFactory can build Watchers from a list of files to be watched for changes
type WatcherFactory func(paths []string) (Watcher, error)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Start watches a set of files for changes, and calls `onChange`
	// on each file change.
	Start(ctx context.Context, onChange func([]string)) error
}

// fsWatcher uses inotify to watch for changes and implements
// the Watcher interface
type fsWatcher struct {
	watcher *fsnotify.Watcher
	files   map[string]bool
}

// NewWatcher creates a new Watcher on a list of files.
func NewWatcher(paths []string) (Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrapf(err, "creating watcher")
	}

	files := map[string]bool{}

	sort.Strings(paths)
	for _, p := range paths {
		files[p] = true
		logrus.Infof("Added watch for %s", p)

		if err := w.Add(p); err != nil {
			w.Close()
			return nil, errors.Wrapf(err, "adding watch for %s", p)
		}

		if err := w.Add(filepath.Dir(p)); err != nil {
			w.Close()
			return nil, errors.Wrapf(err, "adding watch for %s", p)
		}
	}

	logrus.Info("Watch is ready")
	return &fsWatcher{
		watcher: w,
		files:   files,
	}, nil
}

// Start watches a set of files for changes, and calls `onChange`
// on each file change.
func (f *fsWatcher) Start(ctx context.Context, onChange func([]string)) error {
	changedPaths := map[string]bool{}

	timer := time.NewTimer(1<<63 - 1) // Forever
	defer timer.Stop()

	for {
		select {
		case ev := <-f.watcher.Events:
			if ev.Op == fsnotify.Chmod {
				continue // TODO(dgageot): VSCode seems to chmod randomly
			}
			if !f.files[ev.Name] {
				continue // File is not directly watched. Maybe its parent is
			}
			timer.Reset(quietPeriod)
			logrus.Infof("Change: %s", ev)
			changedPaths[ev.Name] = true
		case err := <-f.watcher.Errors:
			return errors.Wrap(err, "watch error")
		case <-timer.C:
			changes := sortedPaths(changedPaths)
			changedPaths = map[string]bool{}
			onChange(changes)
		case <-ctx.Done():
			f.watcher.Close()
			return nil
		}
	}
}

func sortedPaths(changedPaths map[string]bool) []string {
	var paths []string

	for path := range changedPaths {
		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths
}
