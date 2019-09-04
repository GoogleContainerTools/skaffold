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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNoTestDependencies(t *testing.T) {
	runCtx := &runcontext.RunContext{}

	deps, err := NewTester(runCtx).TestDependencies()

	testutil.CheckErrorAndDeepEqual(t, false, err, 0, len(deps))
}

func TestTestDependencies(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

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

	deps, err := NewTester(runCtx).TestDependencies()

	expectedDeps := tmpDir.Paths("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedDeps, deps)
}

func TestWrongPattern(t *testing.T) {
	runCtx := &runcontext.RunContext{
		Cfg: latest.Pipeline{
			Test: []*latest.TestCase{
				{StructureTests: []string{"[]"}},
			},
		},
	}

	tester := NewTester(runCtx)

	_, err := tester.TestDependencies()
	testutil.CheckError(t, true, err)

	err = tester.Test(context.Background(), ioutil.Discard, nil)
	testutil.CheckError(t, true, err)
}

func TestNoTest(t *testing.T) {
	runCtx := &runcontext.RunContext{}

	err := NewTester(runCtx).Test(context.Background(), ioutil.Discard, nil)

	testutil.CheckError(t, false, err)
}

func TestTestSuccess(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Touch("tests/test1.yaml", "tests/test2.yaml", "test3.yaml")

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRun("container-structure-test test -v warn --image TAG --config "+tmpDir.Path("tests/test1.yaml")+" --config "+tmpDir.Path("tests/test2.yaml")).
			AndRun("container-structure-test test -v warn --image TAG --config "+tmpDir.Path("test3.yaml")))

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
				},
			},
		}

		err := NewTester(runCtx).Test(context.Background(), ioutil.Discard, []build.Artifact{{
			ImageName: "image",
			Tag:       "TAG",
		}})

		t.CheckError(false, err)
	})
}

func TestTestFailure(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("test.yaml")

		t.Override(&util.DefaultExecCommand, testutil.CmdRunErr(
			"container-structure-test test -v warn --image broken-image --config "+tmpDir.Path("test.yaml"),
			errors.New("FAIL"),
		))

		runCtx := &runcontext.RunContext{
			WorkingDir: tmpDir.Root(),
			Cfg: latest.Pipeline{
				Test: []*latest.TestCase{
					{
						ImageName:      "broken-image",
						StructureTests: []string{"test.yaml"},
					},
				},
			},
		}

		err := NewTester(runCtx).Test(context.Background(), ioutil.Discard, []build.Artifact{{}})
		t.CheckError(true, err)
	})
}
