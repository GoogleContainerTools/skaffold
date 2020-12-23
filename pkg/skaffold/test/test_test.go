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

package test

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNoTestDependencies(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) { return nil, nil })

		cfg := &mockConfig{}
		deps, err := NewTester(cfg, func(imageName string) (bool, error) { return true, nil }).TestDependencies()

		t.CheckNoError(err)
		t.CheckEmpty(deps)
	})
}

func TestTestDependencies(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{
				{StructureTests: []string{"./tests/*"}},
				{},
				{StructureTests: []string{"test3.yaml"}},
			},
		}
		deps, err := NewTester(cfg, func(imageName string) (bool, error) { return true, nil }).TestDependencies()

		expectedDeps := tmpDir.Paths("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")
		t.CheckNoError(err)
		t.CheckDeepEqual(expectedDeps, deps)
	})
}

func TestWrongPattern(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		cfg := &mockConfig{
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"[]"},
			}},
		}

		tester := NewTester(cfg, func(imageName string) (bool, error) { return true, nil })

		_, err := tester.TestDependencies()
		t.CheckError(true, err)

		err = tester.Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})
		t.CheckError(true, err)
	})
}

func TestNoTest(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		cfg := &mockConfig{}

		tester := NewTester(cfg, func(imageName string) (bool, error) { return true, nil })
		err := tester.Test(context.Background(), ioutil.Discard, nil)

		t.CheckNoError(err)
	})
}

func TestIgnoreDockerNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
			return nil, errors.New("not found")
		})

		tester := NewTester(&mockConfig{}, func(imageName string) (bool, error) { return true, nil })

		t.CheckNil(tester)
	})
}

func TestTestSuccess(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("tests/test1.yaml")+" --config "+tmpDir.Path("tests/test2.yaml")).
			AndRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("test3.yaml")))

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{
				{
					ImageName:      "image",
					StructureTests: []string{"./tests/*"},
				},
				{},
				{
					ImageName:      "image",
					StructureTests: []string{"test3.yaml"},
				},
				{
					// This is image is not built so it won't be tested.
					ImageName:      "not-built",
					StructureTests: []string{"./tests/*"},
				},
			},
		}

		imagesAreLocal := true
		err := NewTester(cfg, func(imageName string) (bool, error) { return imagesAreLocal, nil }).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})

		t.CheckNoError(err)
	})
}

func TestTestSuccessRemoteImage(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Touch("test.yaml").Chdir()
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image image:tag --config test.yaml"))
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
			return fakeLocalDaemon(&testutil.FakeAPIClient{}), nil
		})

		cfg := &mockConfig{
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
			}},
		}

		imagesAreLocal := false
		err := NewTester(cfg, func(imageName string) (bool, error) { return imagesAreLocal, nil }).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})

		t.CheckNoError(err)
	})
}

func TestTestFailureRemoteImage(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Touch("test.yaml").Chdir()
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image image:tag --config test.yaml"))
		t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
			return fakeLocalDaemon(&testutil.FakeAPIClient{ErrImagePull: true}), nil
		})

		cfg := &mockConfig{
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
			}},
		}

		imagesAreLocal := false
		err := NewTester(cfg, func(imageName string) (bool, error) { return imagesAreLocal, nil }).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})

		t.CheckErrorContains(`unable to docker pull image "image:tag"`, err)
	})
}

func TestTestFailure(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Touch("test.yaml").Chdir()

		t.Override(&util.DefaultExecCommand, testutil.CmdRunErr(
			"container-structure-test test -v warn --image broken-image:tag --config test.yaml",
			errors.New("FAIL"),
		))

		cfg := &mockConfig{
			tests: []*latest.TestCase{
				{
					ImageName:      "broken-image",
					StructureTests: []string{"test.yaml"},
				},
			},
		}

		err := NewTester(cfg, func(imageName string) (bool, error) { return true, nil }).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "broken-image",
			Tag:       "broken-image:tag",
		}})
		t.CheckError(true, err)
	})
}

func TestTestMuted(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("test.yaml")))

		cfg := &mockConfig{
			workingDir: tmpDir.Root(),
			tests: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"test.yaml"},
			}},
			muted: config.Muted{
				Phases: []string{"test"},
			},
		}

		var buf bytes.Buffer
		err := NewTester(cfg, func(imageName string) (bool, error) { return true, nil }).Test(context.Background(), &buf, []build.Artifact{{
			ImageName: "image",
			Tag:       "image:tag",
		}})

		t.CheckNoError(err)
		t.CheckContains("- writing logs to "+filepath.Join(os.TempDir(), "skaffold", "test.log"), buf.String())
	})
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
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
