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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHasChanged(t *testing.T) {
	var tests = []struct {
		description     string
		setup           func(tmp *testutil.TempDir)
		update          func(tmp *testutil.TempDir)
		expectedChanged bool
	}{
		{
			description: "no file",
			setup:       func(*testutil.TempDir) {},
			update:      func(*testutil.TempDir) {},
		},
		{
			description:     "added",
			setup:           func(*testutil.TempDir) {},
			update:          func(tmp *testutil.TempDir) { tmp.Write("added", "") },
			expectedChanged: true,
		},
		{
			description:     "removed",
			setup:           func(tmp *testutil.TempDir) { tmp.Write("removed", "") },
			update:          func(tmp *testutil.TempDir) { tmp.Remove("removed") },
			expectedChanged: true,
		},
		{
			description:     "modified",
			setup:           func(tmp *testutil.TempDir) { tmp.Write("file", "") },
			update:          func(tmp *testutil.TempDir) { tmp.Chtimes("file", time.Now().Add(2*time.Second)) },
			expectedChanged: true,
		},
		{
			description: "removed and added",
			setup:       func(tmp *testutil.TempDir) { tmp.Write("removed", "") },
			update: func(tmp *testutil.TempDir) {
				tmp.Remove("removed").Write("added", "").Chtimes("added", time.Now().Add(2*time.Second))
			},
			expectedChanged: true,
		},
		{
			description:     "ignore modified directory",
			setup:           func(tmp *testutil.TempDir) { tmp.Mkdir("dir") },
			update:          func(tmp *testutil.TempDir) { tmp.Chtimes("dir", time.Now().Add(2*time.Second)) },
			expectedChanged: false,
		},
		{
			description:     "broken symlink is handled",
			setup:           func(tmp *testutil.TempDir) { tmp.WriteSymlink("symlink", "") },
			update:          func(tmp *testutil.TempDir) { tmp.Remove("symlink") },
			expectedChanged: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, tearDown := testutil.NewTempDir(t)
			defer tearDown()

			test.setup(tmpDir)
			prev, err := stat(tmpDir.List)
			if err != nil {
				t.Fatal("Unable to setup test directory", err)
			}

			test.update(tmpDir)
			curr, err := stat(tmpDir.List)
			if err != nil {
				t.Fatal("Unable to update test directory", err)
			}

			changed := hasChanged(prev, curr)

			testutil.CheckDeepEqual(t, test.expectedChanged, changed)
		})
	}
}
