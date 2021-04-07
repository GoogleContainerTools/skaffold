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

package custom

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/testutil/event"
)

func TestNewCustomTestRunner(t *testing.T) {
	testutil.Run(t, "Testing new custom test runner", func(t *testutil.T) {
		if runtime.GOOS == Windows {
			t.Override(&util.DefaultExecCommand, testutil.CmdRun("cmd.exe /C echo Running Custom Test command."))
		} else {
			t.Override(&util.DefaultExecCommand, testutil.CmdRun("sh -c echo Running Custom Test command."))
		}
		tmpDir := t.NewTempDir().Touch("test.yaml")

		custom := latest.CustomTest{
			Command:        "echo Running Custom Test command.",
			TimeoutSeconds: 10,
			Dependencies: &latest.CustomTestDependencies{
				Paths:  []string{"**"},
				Ignore: []string{"b*"},
			},
		}

		testCase := &latest.TestCase{
			ImageName:   "image",
			CustomTests: []latest.CustomTest{custom},
		}

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests:      []*latest.TestCase{testCase},
		}
		testEvent.InitializeState([]latest.Pipeline{{}})

		testRunner, err := New(cfg, testCase.ImageName, cfg.workingDir, custom)
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})

		t.CheckNoError(err)
	})
}

func TestCustomCommandError(t *testing.T) {
	tests := []struct {
		description        string
		custom             latest.CustomTest
		shouldErr          bool
		expectedCmd        string
		expectedWindowsCmd string
		expectedError      string
	}{
		{
			description: "Non zero exit",
			custom: latest.CustomTest{
				Command: "exit 20",
			},
			shouldErr:          true,
			expectedCmd:        "sh -c exit 20",
			expectedWindowsCmd: "cmd.exe /C exit 20",
			expectedError:      "exit status 20",
		},
		{
			description: "Command timed out",
			custom: latest.CustomTest{
				Command:        "sleep 20",
				TimeoutSeconds: 2,
			},
			shouldErr:          true,
			expectedCmd:        "sh -c sleep 20",
			expectedWindowsCmd: "cmd.exe /C sleep 20",
			expectedError:      "context deadline exceeded",
		},
	}
	for _, test := range tests {
		testutil.Run(t, "Testing new custom test runner", func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch("test.yaml")
			command := test.expectedCmd
			if runtime.GOOS == Windows {
				command = test.expectedWindowsCmd
			}
			t.Override(&util.DefaultExecCommand, testutil.CmdRunErr(command, fmt.Errorf(test.expectedError)))

			testCase := &latest.TestCase{
				ImageName:   "image",
				CustomTests: []latest.CustomTest{test.custom},
			}

			cfg := &mockConfig{
				workingDir: tmpDir.Root(),
				tests:      []*latest.TestCase{testCase},
			}
			testEvent.InitializeState([]latest.Pipeline{{}})

			testRunner, err := New(cfg, testCase.ImageName, cfg.workingDir, test.custom)
			t.CheckNoError(err)
			err = testRunner.Test(context.Background(), ioutil.Discard, []build.Artifact{{
				ImageName: "image",
				Tag:       "image:tag",
			}})

			// TODO(modali): Update the logic to check for error code instead of error string.
			t.CheckError(test.shouldErr, err)
			if test.expectedError != "" {
				t.CheckErrorContains(test.expectedError, err)
			}
		})
	}
}

func TestTestDependenciesCommand(t *testing.T) {
	testutil.Run(t, "Testing new custom test runner", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")

		custom := latest.CustomTest{
			Command: "echo Hello!",
			Dependencies: &latest.CustomTestDependencies{
				Command: "echo [\"file1\",\"file2\",\"file3\"]",
			},
		}

		testCase := &latest.TestCase{
			ImageName:   "image",
			CustomTests: []latest.CustomTest{custom},
		}

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests:      []*latest.TestCase{testCase},
		}
		testEvent.InitializeState([]latest.Pipeline{{}})

		if runtime.GOOS == Windows {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				"cmd.exe /C echo [\"file1\",\"file2\",\"file3\"]",
				"[\"file1\",\"file2\",\"file3\"]",
			))
		} else {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				"sh -c echo [\"file1\",\"file2\",\"file3\"]",
				"[\"file1\",\"file2\",\"file3\"]",
			))
		}

		expected := []string{"file1", "file2", "file3"}
		testRunner, err := New(cfg, testCase.ImageName, cfg.workingDir, custom)
		t.CheckNoError(err)
		deps, err := testRunner.TestDependencies()

		t.CheckNoError(err)
		t.CheckDeepEqual(expected, deps)
	})
}

func TestTestDependenciesPaths(t *testing.T) {
	tests := []struct {
		description string
		ignore      []string
		paths       []string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "watch everything",
			paths:       []string{"."},
			expected:    []string{"bar", filepath.FromSlash("baz/file"), "foo"},
		},
		{
			description: "watch nothing",
		},
		{
			description: "ignore some paths",
			paths:       []string{"."},
			ignore:      []string{"b*"},
			expected:    []string{"foo"},
		},
		{
			description: "glob",
			paths:       []string{"**"},
			expected:    []string{"bar", filepath.FromSlash("baz/file"), "foo"},
		},
		{
			description: "error",
			paths:       []string{"unknown"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Directory structure:
			//   foo
			//   bar
			// - baz
			//     file
			tmpDir := t.NewTempDir().
				Touch("foo", "bar", "baz/file")

			custom := latest.CustomTest{
				Command: "echo Hello!",
				Dependencies: &latest.CustomTestDependencies{
					Paths:  test.paths,
					Ignore: test.ignore,
				},
			}

			testCase := &latest.TestCase{
				ImageName:   "image",
				CustomTests: []latest.CustomTest{custom},
			}

			cfg := &mockConfig{
				workingDir: tmpDir.Root(),
				tests:      []*latest.TestCase{testCase},
			}
			testEvent.InitializeState([]latest.Pipeline{{}})

			testRunner, err := New(cfg, testCase.ImageName, cfg.workingDir, custom)
			t.CheckNoError(err)
			deps, err := testRunner.TestDependencies()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deps)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		description string
		tag         string
		testContext string
		environ     []string
		expected    []string
	}{

		{
			description: "make sure tags are correct",
			tag:         "gcr.io/image/tag:mytag",
			environ:     nil,
			testContext: "/some/path",
			expected:    []string{"IMAGE=gcr.io/image/tag:mytag", "TEST_CONTEXT=/some/path"},
		}, {
			description: "make sure environ is correctly applied",
			tag:         "gcr.io/image/tag:anothertag",
			environ:     []string{"PATH=/path", "HOME=/root"},
			testContext: "/some/path",
			expected:    []string{"IMAGE=gcr.io/image/tag:anothertag", "TEST_CONTEXT=/some/path", "PATH=/path", "HOME=/root"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.environ })
			t.Override(&testContext, func(string) (string, error) { return test.testContext, nil })
			tmpDir := t.NewTempDir().Touch("test.yaml")

			custom := latest.CustomTest{
				Command: "echo Running Custom Test command.",
			}

			testCase := &latest.TestCase{
				ImageName:   "image",
				CustomTests: []latest.CustomTest{custom},
			}

			cfg := &mockConfig{
				workingDir: tmpDir.Root(),
				tests:      []*latest.TestCase{testCase},
			}
			testEvent.InitializeState([]latest.Pipeline{{}})

			testRunner, err := New(cfg, testCase.ImageName, cfg.workingDir, custom)
			t.CheckNoError(err)
			actual, err := testRunner.getEnv([]build.Artifact{{
				ImageName: "image",
				Tag:       test.tag,
			}})

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	tests                 []*latest.TestCase
}
