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

package structure

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

func TestNewRunner(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&cluster.FindMinikubeBinary, func(context.Context) (string, semver.Version, error) {
			return "", semver.Version{}, errors.New("not found")
		})

		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("test.yaml")))

		testCase := &latest.TestCase{
			ImageName:      "image",
			Workspace:      tmpDir.Root(),
			StructureTests: []string{"test.yaml"},
		}
		cfg := &mockConfig{tests: []*latest.TestCase{testCase}}

		testEvent.InitializeState([]latest.Pipeline{{}})

		testRunner, err := New(context.Background(), cfg, testCase, true)
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), io.Discard, "image:tag")
		t.CheckNoError(err)
	})
}

func TestIgnoreDockerNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
			return nil, errors.New("not found")
		})

		testCase := &latest.TestCase{
			ImageName:      "image",
			Workspace:      tmpDir.Root(),
			StructureTests: []string{"test.yaml"},
		}
		cfg := &mockConfig{tests: []*latest.TestCase{testCase}}

		testRunner, err := New(context.Background(), cfg, testCase, true)
		t.CheckError(true, err)
		t.CheckNil(testRunner)
	})
}

func TestCustomParams(t *testing.T) {
	testCases := []struct {
		structureTestArgs []string
		expectedExtras    string
	}{
		{
			structureTestArgs: []string{"--driver=tar", "--force", "-q", "--save"},
			expectedExtras:    "--driver=tar --force -q --save",
		},
		{
			structureTestArgs: []string{},
			expectedExtras:    "",
		},
		{
			structureTestArgs: nil,
			expectedExtras:    "",
		},
	}

	for _, tc := range testCases {
		testutil.Run(t, "", func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch("test.yaml")
			t.Override(&cluster.FindMinikubeBinary, func(context.Context) (string, semver.Version, error) {
				return "", semver.Version{}, errors.New("not found")
			})

			expected := "container-structure-test test -v warn --image image:tag --config " + tmpDir.Path("test.yaml")
			if len(tc.expectedExtras) > 0 {
				expected += " " + tc.expectedExtras
			}
			t.Override(&util.DefaultExecCommand, testutil.CmdRun(expected))

			testCase := &latest.TestCase{
				ImageName:         "image",
				Workspace:         tmpDir.Root(),
				StructureTests:    []string{"test.yaml"},
				StructureTestArgs: tc.structureTestArgs,
			}
			cfg := &mockConfig{tests: []*latest.TestCase{testCase}}

			testEvent.InitializeState([]latest.Pipeline{{}})

			testRunner, err := New(context.Background(), cfg, testCase, true)
			t.CheckNoError(err)
			err = testRunner.Test(context.Background(), io.Discard, "image:tag")
			t.CheckNoError(err)
		})
	}
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	tests                 []*latest.TestCase
	muted                 config.Muted
}

func (c *mockConfig) Muted() config.Muted { return c.muted }

func (c *mockConfig) TestCases() []*latest.TestCase { return c.tests }
