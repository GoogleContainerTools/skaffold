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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestWatch(t *testing.T) {
	watchers := []string{"mtime", "fsnotify"}
	var tests = []struct {
		description     string
		createFiles     []string
		watchFiles      []string
		writes          []string
		deletes         []string
		expectedChanges []string
		shouldErr       bool
	}{
		{
			description: "watch unknown file",
			createFiles: []string{"a"},
			watchFiles:  []string{"a", "b"},
			shouldErr:   true,
		},
		{
			description:     "write files",
			createFiles:     []string{"a", "b", "c"},
			watchFiles:      []string{"a", "b", "c"},
			writes:          []string{"a", "b"},
			expectedChanges: []string{"a", "b"},
		},
		{
			description:     "ignore file",
			createFiles:     []string{"a", "b"},
			watchFiles:      []string{"a"},
			writes:          []string{"a", "b"},
			expectedChanges: []string{"a"},
		},
	}

	for _, watcher := range watchers {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s %s", test.description, watcher), func(t *testing.T) {
				os.Setenv("SKAFFOLD_FILE_WATCHER", watcher)
				defer os.Setenv("SKAFFOLD_FILE_WATCHER", "")
				tmp, teardown := testutil.TempDir(t)
				defer teardown()

				for _, p := range prependParentDir(tmp, test.createFiles) {
					write(t, p, "")
				}

				watcher, err := NewWatcher(prependParentDir(tmp, test.watchFiles))
				if err == nil && test.shouldErr {
					t.Errorf("Expected error, but returned none")
					return
				}
				if err != nil && !test.shouldErr {
					t.Errorf("Unexpected error: %s", err)
					return
				}
				if err != nil && test.shouldErr {
					return
				}

				ctx, cancel := context.WithCancel(context.Background())

				defer cancel()
				go watcher.Start(ctx, ioutil.Discard, func(actual []string) error {

					expected := prependParentDir(tmp, test.expectedChanges)

					if diff := cmp.Diff(expected, actual); diff != "" {
						t.Errorf("Expected %+v, Actual %+v", expected, actual)
					}

					return nil
				})

				for _, p := range prependParentDir(tmp, test.writes) {
					write(t, p, "CONTENT")
				}
			})
		}
	}
}
func write(t *testing.T, path string, content string) {
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing file: %s", err)
	}
}

func prependParentDir(parentDir string, paths []string) []string {
	var list []string
	for _, path := range paths {
		list = append(list, filepath.Join(parentDir, path))
	}
	return list
}
