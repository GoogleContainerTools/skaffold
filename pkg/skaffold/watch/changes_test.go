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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	yesterday, _ = time.Parse(
		time.RFC3339,
		"1991-11-26T22:08:41+00:00")
	today, _ = time.Parse(
		time.RFC3339,
		"1991-11-27T22:08:41+00:00")
)

func TestEvents(t *testing.T) {
	var tests = []struct {
		description   string
		prev, current fileMap
		expected      Events
	}{
		{
			description: "added, modified, and deleted files",
			prev: map[string]time.Time{
				"a": yesterday,
				"b": yesterday,
			},
			current: map[string]time.Time{
				"a": today,
				"c": today,
			},
			expected: Events{
				Added:    []string{"c"},
				Modified: []string{"a"},
				Deleted:  []string{"b"},
			},
		},
		{
			description: "no changes",
			prev: map[string]time.Time{
				"a": today,
				"b": today,
			},
			current: map[string]time.Time{
				"a": today,
				"b": today,
			},
			expected: Events{},
		},
		{
			description: "added all",
			prev:        map[string]time.Time{},
			current: map[string]time.Time{
				"a": today,
				"b": today,
				"c": today,
			},
			expected: Events{Added: []string{"a", "b", "c"}},
		},
		{
			description: "deleted all",
			prev: map[string]time.Time{
				"a": today,
				"b": today,
				"c": today,
			},
			current:  map[string]time.Time{},
			expected: Events{Deleted: []string{"a", "b", "c"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			testutil.CheckDeepEqual(t, test.expected, events(test.prev, test.current))
		})
	}
}

func TestStat(t *testing.T) {
	var tests = []struct {
		description string
		setup       func(folder *testutil.TempDir)
		expected    fileMap
		shouldErr   bool
	}{
		{
			description: "stat files",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			test.setup(folder)
			list, _ := folder.List()

			actual, err := stat(folder.List)
			testutil.CheckError(t, test.shouldErr, err)
			checkListInMap(t, list, actual)
		})
	}
}

func TestStatNotExist(t *testing.T) {
	var tests = []struct {
		description string
		setup       func(folder *testutil.TempDir)
		deps        []string
		depsErr     error
		expected    fileMap
		shouldErr   bool
	}{
		{
			description: "no error when deps returns nonexisting file",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
			deps: []string{"file/that/doesnt/exist/anymore"},
		},
		{
			description: "deps function error",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
			deps:      []string{"file/that/doesnt/exist/anymore"},
			depsErr:   fmt.Errorf(""),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			test.setup(folder)

			_, err := stat(func() ([]string, error) { return test.deps, test.depsErr })
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func checkListInMap(t *testing.T, list []string, m fileMap) {
	for _, f := range list {
		if _, ok := m[f]; !ok {
			t.Errorf("File %s not in map", f)
		}
	}
	if len(list) != len(m) {
		t.Errorf("List and map length differ %s, %s", list, m)
	}
}
