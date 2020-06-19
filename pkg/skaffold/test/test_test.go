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
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNoTestDependencies(t *testing.T) {
	runCtx := &runcontext.RunContext{}

	deps, err := NewTester(runCtx, true).TestDependencies()

	testutil.CheckErrorAndDeepEqual(t, false, err, 0, len(deps))
}

func TestTestDependencies(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	tmpDir.Touch("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")

	runCtx := &runcontext.RunContext{
		WorkingDir: tmpDir.Root(),
		Cfg: latest.Pipeline{
			Test: []*latest.TestCase{
				{StructureTests: []string{"./tests/*"}},
				{},
				{StructureTests: []string{"test3.yaml"}},
			},
		},
	}

	deps, err := NewTester(runCtx, true).TestDependencies()

	expectedDeps := tmpDir.Paths("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedDeps, deps)
}

func TestWrongPattern(t *testing.T) {
	runCtx := &runcontext.RunContext{
		Cfg: latest.Pipeline{
			Test: []*latest.TestCase{{
				ImageName:      "image",
				StructureTests: []string{"[]"},
			}},
		},
	}

	tester := NewTester(runCtx, true)

	_, err := tester.TestDependencies()
	testutil.CheckError(t, true, err)

	err = tester.Test(context.Background(), ioutil.Discard, []build.Artifact{{
		ImageName: "image",
		Tag:       "image:tag",
	}})
	testutil.CheckError(t, true, err)
}

func TestNoTest(t *testing.T) {
	runCtx := &runcontext.RunContext{}

	err := NewTester(runCtx, true).Test(context.Background(), ioutil.Discard, nil)

	testutil.CheckError(t, false, err)
}

func TestIgnoreDockerNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return nil, errors.New("not found")
		})

		tester := NewTester(&runcontext.RunContext{}, true)

		t.CheckNil(tester)
	})
}

func TestTestSuccess(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("tests/test1.yaml")+" --config "+tmpDir.Path("tests/test2.yaml")).
			AndRun("container-structure-test test -v warn --image image:tag --config "+tmpDir.Path("test3.yaml")))

		runCtx := &runcontext.RunContext{
			WorkingDir: tmpDir.Root(),
			Cfg: latest.Pipeline{
				Test: []*latest.TestCase{
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
			},
		}

		imagesAreLocal := true
		err := NewTester(runCtx, imagesAreLocal).Test(context.Background(), ioutil.Discard, []build.Artifact{{
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
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil), nil
		})

		runCtx := &runcontext.RunContext{
			Cfg: latest.Pipeline{
				Test: []*latest.TestCase{{
					ImageName:      "image",
					StructureTests: []string{"test.yaml"},
				}},
			},
		}

		imagesAreLocal := false
		err := NewTester(runCtx, imagesAreLocal).Test(context.Background(), ioutil.Discard, []build.Artifact{{
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
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return docker.NewLocalDaemon(&testutil.FakeAPIClient{ErrImagePull: true}, nil, false, nil), nil
		})

		runCtx := &runcontext.RunContext{
			Cfg: latest.Pipeline{
				Test: []*latest.TestCase{{
					ImageName:      "image",
					StructureTests: []string{"test.yaml"},
				}},
			},
		}

		imagesAreLocal := false
		err := NewTester(runCtx, imagesAreLocal).Test(context.Background(), ioutil.Discard, []build.Artifact{{
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

		runCtx := &runcontext.RunContext{
			Cfg: latest.Pipeline{
				Test: []*latest.TestCase{
					{
						ImageName:      "broken-image",
						StructureTests: []string{"test.yaml"},
					},
				},
			},
		}

		err := NewTester(runCtx, true).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "broken-image",
			Tag:       "broken-image:tag",
		}})
		t.CheckError(true, err)
	})
}
