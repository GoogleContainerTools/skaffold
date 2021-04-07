// +build !windows
/*
Copyright 2021 The Skaffold Authors

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

package fsnotify

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"

	"github.com/rjeczalik/notify"
)

func TestHidden(t *testing.T) {
	tests := []struct {
		description string
		filename    string
		expected    bool
	}{
		{
			description: "ignore hidden files at some path",
			filename:    "/some/.hidden.swp",
			expected:    true,
		},
		{
			description: "ignore hidden files at root",
			filename:    "/.hidden.swp",
			expected:    true,
		},
		{
			description: "ignore files inside hidden dir",
			filename:    "/.hidden/somefile.txt",
			expected:    true,
		},
		{
			description: "don't ignore abs files which are not hidden",
			filename:    "/not-hidden/somefile.txt",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tr := &Trigger{
				Interval: 10,
			}
			t.CheckDeepEqual(test.expected, tr.hidden(test.filename))
		})
	}
}

type mockEvent struct {
	file string
}

func (m mockEvent) Path() string {
	return m.file
}

func (m mockEvent) Event() notify.Event {
	return notify.FSEventsModified
}

func (m mockEvent) Sys() interface{} {
	return m
}
