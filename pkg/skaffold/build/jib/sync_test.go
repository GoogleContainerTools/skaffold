/*
Copyright 2020 The Skaffold Authors

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

package jib

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetSyncMapFromSystem(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Touch("dep1", "dir/dep2")
	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dir/dep2")

	dep1Time := getFileTime(dep1, t)
	dep2Time := getFileTime(dep2, t)

	dep1Target := "/target/dep1"
	dep2Target := "/target/anotherDir/dep2"

	tests := []struct {
		description string
		stdout      string
		shouldErr   bool
		expected    *SyncMap
	}{
		{
			description: "empty",
			stdout:      "",
			shouldErr:   true,
			expected:    nil,
		},
		{
			description: "old style marker",
			stdout:      "BEGIN JIB JSON\n{}",
			shouldErr:   false,
			expected:    &SyncMap{},
		},
		{
			description: "bad marker",
			stdout:      "BEGIN JIB JSON: BAD/1\n{}",
			shouldErr:   true,
			expected:    nil,
		},
		{
			description: "direct only",
			stdout: "BEGIN JIB JSON: SYNCMAP/1\n" +
				fmt.Sprintf(`{"direct":[{"src":"%s","dest":"%s"}]}`, dep1, dep1Target),
			shouldErr: false,
			expected: &SyncMap{
				dep1: SyncEntry{
					[]string{dep1Target},
					dep1Time,
					true,
				},
			},
		},
		{
			description: "generated only",
			stdout: "BEGIN JIB JSON: SYNCMAP/1\n" +
				fmt.Sprintf(`{"generated":[{"src":"%s","dest":"%s"}]}`, dep1, dep1Target),
			shouldErr: false,
			expected: &SyncMap{
				dep1: SyncEntry{
					[]string{dep1Target},
					dep1Time,
					false,
				},
			},
		},
		{
			description: "generated and direct",
			stdout: "BEGIN JIB JSON: SYNCMAP/1\n" +
				fmt.Sprintf(`{"direct":[{"src":"%s","dest":"%s"}],"generated":[{"src":"%s","dest":"%s"}]}"`, dep1, dep1Target, dep2, dep2Target),
			shouldErr: false,
			expected: &SyncMap{
				dep1: SyncEntry{
					[]string{dep1Target},
					dep1Time,
					true,
				},
				dep2: SyncEntry{
					Dest:     []string{dep2Target},
					FileTime: dep2Time,
					IsDirect: false,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				"ignored",
				test.stdout,
			))

			results, err := getSyncMapFromSystem(&exec.Cmd{Args: []string{"ignored"}})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, results)
		})
	}
}

func getFileTime(file string, t *testing.T) time.Time {
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("Failed to stat %s", file)
		return time.Time{}
	}
	return info.ModTime()
}
