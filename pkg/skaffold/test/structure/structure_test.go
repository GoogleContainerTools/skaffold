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
	"io/ioutil"
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/testutil/event"
)

func TestNewRunner(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&cluster.FindMinikubeBinary, func() (string, semver.Version, error) { return "", semver.Version{}, errors.New("not found") })

		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("test.yaml")))

		testCase := &latestV1.TestCase{
			ImageName:      "image",
			Workspace:      tmpDir.Root(),
			StructureTests: []string{"test.yaml"},
		}
		cfg := &mockConfig{tests: []*latestV1.TestCase{testCase}}

		testEvent.InitializeState([]latestV1.Pipeline{{}})

		testRunner, err := New(ctx, cfg, testCase, true)
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), ioutil.Discard, "image:tag")
		t.CheckNoError(err)
	})
}

func TestIgnoreDockerNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
			return nil, errors.New("not found")
		})

		testCase := &latestV1.TestCase{
			ImageName:      "image",
			Workspace:      tmpDir.Root(),
			StructureTests: []string{"test.yaml"},
		}
		cfg := &mockConfig{tests: []*latestV1.TestCase{testCase}}

		testRunner, err := New(ctx, cfg, testCase, true)
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
			t.Override(&cluster.FindMinikubeBinary, func() (string, semver.Version, error) { return "", semver.Version{}, errors.New("not found") })

			expected := "container-structure-test test -v warn --image image:tag --config " + tmpDir.Path("test.yaml")
			if len(tc.expectedExtras) > 0 {
				expected += " " + tc.expectedExtras
			}
			t.Override(&util.DefaultExecCommand, testutil.CmdRun(expected))

			testCase := &latestV1.TestCase{
				ImageName:         "image",
				Workspace:         tmpDir.Root(),
				StructureTests:    []string{"test.yaml"},
				StructureTestArgs: tc.structureTestArgs,
			}
			cfg := &mockConfig{tests: []*latestV1.TestCase{testCase}}

			testEvent.InitializeState([]latestV1.Pipeline{{}})

			testRunner, err := New(ctx, cfg, testCase, true)
			t.CheckNoError(err)
			err = testRunner.Test(context.Background(), ioutil.Discard, "image:tag")
			t.CheckNoError(err)
		})
	}
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	tests                 []*latestV1.TestCase
	muted                 config.Muted
}

func (c *mockConfig) Muted() config.Muted { return c.muted }

func (c *mockConfig) TestCases() []*latestV1.TestCase { return c.tests }
