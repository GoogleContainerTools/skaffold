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
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//TODO(@r2d4): Figure out best UX to support configuring this blacklist
var ignoredFiles = []string{}
var ignoredDirs = []string{"vendor", ".git"}

// FileChangedFn is a function called when files where changed.
type FileChangedFn func(changes []string) error

// FileWatcher watches for file changes.
type FileWatcher interface {
	Run(ctx context.Context, callback FileChangedFn) error
}

type fileMap map[string]os.FileInfo

type fileWatcher struct {
	dirs         []string
	pollInterval time.Duration
	files        fileMap
}

// NewFileWatcher creates a FileWatcher for a list of directories.
func NewFileWatcher(dirs []string, pollInterval time.Duration) (FileWatcher, error) {
	files, err := walk(dirs)
	if err != nil {
		return nil, errors.Wrap(err, "listing files in folders")
	}

	return &fileWatcher{
		dirs:         dirs,
		files:        files,
		pollInterval: pollInterval,
	}, nil
}

func walk(dirs []string) (fileMap, error) {
	m := make(fileMap)

	uniqueDirs := util.UniqueStrSlice(dirs)
	for _, dir := range uniqueDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if isIgnored(info.Name(), info.IsDir()) {
				return filepath.SkipDir
			}

			m[path] = info
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

func computeDiff(prev, curr fileMap) []string {
	var changes []string

	for k, prevV := range prev {
		currV, ok := curr[k]
		if !ok {
			// Deleted
			changes = append(changes, k)
		} else if prevV.ModTime() != currV.ModTime() {
			if currV.IsDir() && currV.IsDir() == prevV.IsDir() {
				// Ignore directory time changes
			} else {
				// Changed mtime
				changes = append(changes, k)
			}
		}
	}
	for k := range curr {
		if _, ok := prev[k]; !ok {
			// Created
			changes = append(changes, k)
		}
	}

	sort.Strings(changes)
	return changes
}

func isIgnored(path string, isDir bool) bool {
	var files []string
	if isDir {
		files = ignoredDirs
	} else {
		files = ignoredFiles
	}
	for _, i := range files {
		if path == i {
			return true
		}
	}

	return false
}

func (w *fileWatcher) Run(ctx context.Context, callback FileChangedFn) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	var previousDiff []string
	previousFiles := w.files

	for {
		select {
		case <-ticker.C:
			files, err := walk(w.dirs)
			if err != nil {
				return errors.Wrap(err, "listing files in folders")
			}

			changesSinceLastTick := computeDiff(previousFiles, files)
			if len(changesSinceLastTick) > 0 {
				logrus.Debugln("First change detected")
			} else if len(previousDiff) > 0 {
				diff := computeDiff(w.files, files)
				if len(diff) > 0 {
					if err := callback(diff); err != nil {
						return errors.Wrap(err, "watcher callback")
					}
				}

				w.files = files
			}

			previousFiles = files
			previousDiff = changesSinceLastTick
		case <-ctx.Done():
			return nil
		}
	}
}
