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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFileMonitor(t *testing.T) {
	tests := []struct {
		description string
		makeChanges func(folder *testutil.TempDir)
	}{
		{
			description: "file change",
			makeChanges: func(folder *testutil.TempDir) {
				folder.Chtimes("file", time.Now().Add(2*time.Second))
			},
		},
		{
			description: "file delete",
			makeChanges: func(folder *testutil.TempDir) {
				folder.Remove("file")
			},
		},
		{
			description: "file create",
			makeChanges: func(folder *testutil.TempDir) {
				folder.Touch("new")
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch("file")

			monitor := NewMonitor()

			// Register files
			changed := callback{}
			err := monitor.Register(tmpDir.List, changed.call)
			t.CheckNoError(err)
			t.CheckDeepEqual(0, changed.calls())

			test.makeChanges(tmpDir)

			// Verify the Monitor detects a change
			err = monitor.Run(false)
			t.CheckNoError(err)
			t.CheckDeepEqual(1, changed.calls())

			// Verify the Monitor doesn't detect more changes
			err = monitor.Run(false)
			t.CheckNoError(err)
			t.CheckDeepEqual(1, changed.calls())
		})
	}
}

type callback struct {
	events []Events
}

func (c *callback) call(e Events) {
	c.events = append(c.events, e)
}

func (c *callback) calls() int {
	return len(c.events)
}
