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
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WatcherFactory can build Watchers from a list of files to be watched for changes
type WatcherFactory func(paths []string) (Watcher, error)

// Watcher provides a watch trigger for the skaffold pipeline to begin
type Watcher interface {
	// Start watches a set of files for changes, and calls `onChange`
	// on each file change.
	Start(ctx context.Context, out io.Writer, onChange func([]string) error) error
}

// mtimeWatcher uses polling on file mTimes.
type mtimeWatcher struct {
	files map[string]time.Time
}

func (m *mtimeWatcher) Start(ctx context.Context, out io.Writer, onChange func([]string) error) error {
	c := time.NewTicker(2 * time.Second)
	defer c.Stop()

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
	logrus.Info("Starting mtime file watcher.")

	sort.Strings(paths)

	files := map[string]time.Time{}
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, errors.Wrapf(err, "statting file %s", p)
		}
		files[p] = fi.ModTime()
	}

	return &mtimeWatcher{
		files: files,
	}, nil
}

func sortedPaths(changedPaths map[string]bool) []string {
	var paths []string

	for path := range changedPaths {
		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths
}
