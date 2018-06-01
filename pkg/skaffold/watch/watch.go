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
	"context"
	"fmt"
	"io"
	"os"
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
	Start(ctx context.Context, out io.Writer, onChange func([]string) error) error
}

// fsWatcher uses inotify to watch for changes and implements
// the Watcher interface
type fsWatcher struct {
	watcher *fsnotify.Watcher
	files   map[string]bool
}

type mtimeWatcher struct {
	files map[string]time.Time
}

func (m *mtimeWatcher) Start(ctx context.Context, out io.Writer, onChange func([]string) error) error {

	c := time.NewTicker(2 * time.Second)

	changedPaths := map[string]bool{}

	fmt.Fprintln(out, "Watching for changes...")
	for {
		select {
		case <-c.C:
			// add things to changedpaths
			for f := range m.files {
				fi, err := os.Stat(f)
				if err != nil {
					return errors.Wrapf(err, "statting file %s", f)
				}
				mtime, ok := m.files[f]
				if !ok {
					logrus.Warningf("file %s not found.", f)
					continue
				}
				if mtime != fi.ModTime() {
					m.files[f] = fi.ModTime()
					changedPaths[f] = true
				}
			}
			if len(changedPaths) > 0 {
				if err := onChange(sortedPaths(changedPaths)); err != nil {
					return errors.Wrap(err, "change callback")
				}
				logrus.Debugf("Files changed: %v", changedPaths)
				changedPaths = map[string]bool{}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// NewWatcher creates a new Watcher on a list of files.
func NewWatcher(paths []string) (Watcher, error) {
	sort.Strings(paths)

	// Get the watcher type to use, defaulting to mtime.
	watcher := os.Getenv("SKAFFOLD_FILE_WATCHER")
	if watcher == "" {
		watcher = "mtime"
	}

	switch watcher {
	case "mtime":
		logrus.Info("Starting mtime file watcher.")
		files := map[string]time.Time{}
		for _, p := range paths {
			fi, err := os.Stat(p)
			if err != nil {
				return nil, err
			}
			files[p] = fi.ModTime()
		}
		return &mtimeWatcher{
			files: files,
		}, nil
	case "fsnotify":
		logrus.Info("Starting fsnotify file watcher.")
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, errors.Wrapf(err, "creating watcher")
		}

		files := map[string]bool{}

		for _, p := range paths {
			files[p] = true
			logrus.Debugf("Added watch for %s", p)

			if err := w.Add(p); err != nil {
				w.Close()
				return nil, errors.Wrapf(err, "adding watch for %s", p)
			}

			if err := w.Add(filepath.Dir(p)); err != nil {
				w.Close()
				return nil, errors.Wrapf(err, "adding watch for %s", p)
			}
		}
		return &fsWatcher{
			watcher: w,
			files:   files,
		}, nil
	}
	return nil, fmt.Errorf("unknown watch type: %s", watcher)
}

// Start watches a set of files for changes, and calls `onChange`
// on each file change.
func (f *fsWatcher) Start(ctx context.Context, out io.Writer, onChange func([]string) error) error {
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

			if err := onChange(changes); err != nil {
				return errors.Wrap(err, "change callback")
			}

			fmt.Fprintln(out, "Watching for changes...")
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
