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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/testutil/event"
)

func TestNewRunner(t *testing.T) {
	const (
		imageName = "image:tag"
	)

	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image "+imageName+" --config "+tmpDir.Path("test.yaml")))

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
			}},
		}

		testCase := &latest.TestCase{
			ImageName:      "image",
			StructureTests: []string{"test.yaml"},
		}
		testEvent.InitializeState([]latest.Pipeline{{}})

		testRunner, err := New(cfg, cfg.workingDir, testCase, func(imageName string) (bool, error) { return true, nil })
		t.CheckNoError(err)
		err = testRunner.Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})
		t.CheckNoError(err)
	})
}

func TestIgnoreDockerNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
			return nil, errors.New("not found")
		})

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
			}},
		}

		testCase := &latest.TestCase{
			ImageName:      "image",
			StructureTests: []string{"test.yaml"},
		}

		testRunner, err := New(cfg, cfg.workingDir, testCase, func(imageName string) (bool, error) { return true, nil })
		t.CheckError(true, err)
		t.CheckNil(testRunner)
	})
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	tests                 []*latest.TestCase
	muted                 config.Muted
}

func (c *mockConfig) Muted() config.Muted           { return c.muted }
func (c *mockConfig) GetWorkingDir() string         { return c.workingDir }
func (c *mockConfig) TestCases() []*latest.TestCase { return c.tests }
