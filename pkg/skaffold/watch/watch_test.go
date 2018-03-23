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
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"

	"github.com/spf13/afero"
)

func write(t *testing.T, path string) {
	if err := afero.WriteFile(util.Fs, path, []byte(""), 0640); err != nil {
		t.Errorf("writing mock fs file: %s", err)
	}
}

func TestWatch(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var tests = []struct {
		description    string
		watchFiles     []string
		writes         []string
		expectedChange []string
		shouldErr      bool
	}{
		{
			description:    "write file",
			watchFiles:     []string{"testdata/a", "testdata/b", "testdata/c"},
			writes:         []string{"testdata/a", "testdata/b"},
			expectedChange: []string{"testdata/a", "testdata/b"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			watcher, err := NewWatcher(test.watchFiles)
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

			for _, p := range test.writes {
				write(t, p)
			}

			watcher.Start(ctx, func(actual []string) {
				fmt.Println(actual)
				sort.Strings(actual)
				relPaths, err := util.AbsPathToRelativePath(wd, actual)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(relPaths, test.expectedChange) {
					t.Errorf("Expected %+v, Actual %+v", test.expectedChange, actual)
				}

				cancel()
			})
		})
	}
}
