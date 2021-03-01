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
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewCustomTestRunner(t *testing.T) {
	testutil.Run(t, "Testing new custom test runner", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")

		custom := latest.CustomTest{
			Command:        "echo Running Custom Test command.",
			TimeoutSeconds: 10,
			Dependencies: &latest.CustomTestDependencies{
				Command: "echo [\"file1\",\"file2\",\"file3\"]",
				Paths:   []string{"**"},
				Ignore:  []string{"b*"},
			},
		}

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
				CustomTests:    []latest.CustomTest{custom},
			}},
		}

		testRunner, err := New(cfg, cfg.workingDir, custom)
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), ioutil.Discard, nil)

		t.CheckNoError(err)
	})
}

func TestCustomCommandError(t *testing.T) {
	tests := []struct {
		description   string
		custom        latest.CustomTest
		shouldErr     bool
		expectedError string
	}{
		{
			description: "Non zero exit",
			custom: latest.CustomTest{
				Command:        "exit -1",
				TimeoutSeconds: 10,
			},
			shouldErr:     true,
			expectedError: "exit status 255",
		},
		{
			description: "Command timed out",
			custom: latest.CustomTest{
				Command:        "sleep 20",
				TimeoutSeconds: 2,
			},
			shouldErr:     true,
			expectedError: "context deadline exceeded",
		},
	}
	for _, test := range tests {
		testutil.Run(t, "Testing new custom test runner", func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch("test.yaml")

			cfg := &mockConfig{
				workingDir: tmpDir.Root(),
				tests: []*latest.TestCase{{
					ImageName:      "image",
					StructureTests: []string{"test.yaml"},
					CustomTests:    []latest.CustomTest{test.custom},
				}},
			}

			testRunner, err := New(cfg, cfg.workingDir, test.custom)
			t.CheckNoError(err)
			err = testRunner.Test(context.Background(), ioutil.Discard, nil)

			t.CheckError(test.shouldErr, err)
			if test.expectedError != "" {
				t.CheckErrorContains(test.expectedError, err)
			}
		})
	}
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	tests                 []*latest.TestCase
}
