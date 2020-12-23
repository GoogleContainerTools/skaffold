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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetSyncMapFromSystem(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

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
			shouldErr:   true,
			expected:    nil,
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
				fmt.Sprintf(`{"direct":[{"src":"%s","dest":"%s"}]}`, escapeBackslashes(dep1), dep1Target),
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
				fmt.Sprintf(`{"generated":[{"src":"%s","dest":"%s"}]}`, escapeBackslashes(dep1), dep1Target),
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
				fmt.Sprintf(`{"direct":[{"src":"%s","dest":"%s"}],"generated":[{"src":"%s","dest":"%s"}]}"`, escapeBackslashes(dep1), dep1Target, escapeBackslashes(dep2), dep2Target),
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

func TestGetSyncDiff(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	ctx := context.Background()
	workspace := "testworkspace"
	expectedDelete := map[string][]string(nil)

	tmpDir.Touch("build-def", "direct", "dir/generated")

	buildFile := tmpDir.Path("build-def")

	directFile := tmpDir.Path("direct")
	directTarget := []string{"/target/direct"}
	directFileTime := getFileTime(directFile, t)

	generatedFile := tmpDir.Path("dir/generated")
	generatedTarget := []string{"/target/anotherDir/generated"}
	generatedFileTime := getFileTime(generatedFile, t)

	newFile := tmpDir.Path("some/new/file")
	newFileTarget := []string{"/target/some/new/file/place"}

	currSyncMap := SyncMap{
		directFile:    SyncEntry{directTarget, directFileTime, true},
		generatedFile: SyncEntry{generatedTarget, generatedFileTime, false},
	}

	tests := []struct {
		description      string
		artifact         *latest.JibArtifact
		events           filemon.Events
		buildDefinitions []string
		nextSyncMap      SyncMap
		expectedCopy     map[string][]string
		shouldErr        bool
	}{
		{
			description:      "build file changed (nil, nil, nil)",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Modified: []string{buildFile}},
			buildDefinitions: []string{buildFile},
			expectedCopy:     nil,
			shouldErr:        false,
		},
		{
			description:      "something is deleted (nil, nil, nil)",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Deleted: []string{directFile}},
			buildDefinitions: []string{},
			expectedCopy:     nil,
			shouldErr:        false,
		},
		{
			description:      "only direct sync entries changed",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Modified: []string{directFile}},
			buildDefinitions: []string{},
			expectedCopy:     map[string][]string{directFile: directTarget},
			shouldErr:        false,
		},
		{
			description:      "only generated sync entries changed",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Modified: []string{generatedFile}},
			buildDefinitions: []string{},
			nextSyncMap: SyncMap{
				directFile:    SyncEntry{directTarget, directFileTime, true},
				generatedFile: SyncEntry{generatedTarget, time.Now(), false},
			},
			expectedCopy: map[string][]string{generatedFile: generatedTarget},
			shouldErr:    false,
		},
		{
			description:      "generated and direct sync entries changed",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Modified: []string{directFile, generatedFile}},
			buildDefinitions: []string{},
			nextSyncMap: SyncMap{
				directFile:    SyncEntry{directTarget, time.Now(), true},
				generatedFile: SyncEntry{generatedTarget, time.Now(), false},
			},
			expectedCopy: map[string][]string{directFile: directTarget, generatedFile: generatedTarget},
			shouldErr:    false,
		},
		{
			description:      "new file created",
			artifact:         &latest.JibArtifact{},
			events:           filemon.Events{Added: []string{newFile}},
			buildDefinitions: []string{},
			nextSyncMap: SyncMap{
				directFile:    SyncEntry{directTarget, directFileTime, true},
				generatedFile: SyncEntry{generatedTarget, generatedFileTime, false},
				newFile:       SyncEntry{newFileTarget, time.Now(), false},
			},
			expectedCopy: map[string][]string{newFile: newFileTarget},
			shouldErr:    false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getSyncMapFunc, func(_ context.Context, _ string, _ *latest.JibArtifact) (*SyncMap, error) {
				return &test.nextSyncMap, nil
			})
			pk := getProjectKey(workspace, test.artifact)
			t.Override(&watchedFiles, map[projectKey]filesLists{
				pk: {BuildDefinitions: test.buildDefinitions},
			})
			t.Override(&syncLists, map[projectKey]SyncMap{
				pk: currSyncMap,
			})

			toCopy, toDelete, err := GetSyncDiff(ctx, workspace, test.artifact, test.events)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedCopy, toCopy)
			t.CheckDeepEqual(expectedDelete, toDelete)
		})
	}
}

func TestGetSyncDiff_directChecksUpdateFileTime(testing *testing.T) {
	tmpDir := testutil.NewTempDir(testing)

	ctx := context.Background()
	workspace := "testworkspace"
	artifact := &latest.JibArtifact{}

	tmpDir.Touch("direct")

	directFile := tmpDir.Path("direct")
	directTarget := []string{"/target/direct"}
	directFileTime := getFileTime(directFile, testing)

	currSyncMap := SyncMap{
		directFile: SyncEntry{directTarget, directFileTime, true},
	}

	testutil.Run(testing, "Checks on direct files also update file times", func(t *testutil.T) {
		pk := getProjectKey(workspace, artifact)
		t.Override(&getSyncMapFunc, func(_ context.Context, _ string, _ *latest.JibArtifact) (*SyncMap, error) {
			t.Fatal("getSyncMapFunc should not have been called in this test")
			return nil, nil
		})
		t.Override(&watchedFiles, map[projectKey]filesLists{
			pk: {BuildDefinitions: []string{}},
		})
		t.Override(&syncLists, map[projectKey]SyncMap{
			pk: currSyncMap,
		})

		// turns out macOS doesn't exactly set the time you pass to Chtimes so set the time and then read it in.
		tmpDir.Chtimes("direct", time.Now())
		updatedFileTime := getFileTime(directFile, testing)

		_, _, err := GetSyncDiff(ctx, workspace, artifact, filemon.Events{Modified: []string{directFile}})

		t.CheckNoError(err)
		t.CheckDeepEqual(SyncMap{directFile: SyncEntry{directTarget, updatedFileTime, true}}, syncLists[pk])
	})
}

func getFileTime(file string, t *testing.T) time.Time {
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("Failed to stat %s", file)
		return time.Time{}
	}
	return info.ModTime()
}

// for paths that contain "\", they must be escaped in json strings
func escapeBackslashes(path string) string {
	return strings.Replace(path, `\`, `\\`, -1)
}
