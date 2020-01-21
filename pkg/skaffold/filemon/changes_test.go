/*
Copyright 2019 The Skaffold Authors

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
package filemon

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
	tests := []struct {
		description   string
		prev, current FileMap
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
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, events(test.prev, test.current))
		})
	}
}

func TestStat(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("file", "content")

		list, _ := tmpDir.List()
		actual, err := Stat(tmpDir.List)

		t.CheckNoError(err)
		t.CheckDeepEqual(len(list), len(actual))
		for _, f := range list {
			_, present := actual[f]

			t.CheckTrue(present)
		}
	})
}

func TestStatNotExist(t *testing.T) {
	tests := []struct {
		description string
		deps        []string
		depsErr     error
		shouldErr   bool
	}{
		{
			description: "no error when deps returns nonexistent file",
			deps:        []string{"file/that/does/not/exist/anymore"},
		},
		{
			description: "deps function error",
			deps:        []string{"file/that/does/not/exist/anymore"},
			depsErr:     fmt.Errorf(""),
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Write("file", "content")

			_, err := Stat(func() ([]string, error) { return test.deps, test.depsErr })

			t.CheckError(test.shouldErr, err)
		})
	}
}
