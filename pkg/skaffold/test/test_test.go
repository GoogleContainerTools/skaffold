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
	"testing"

	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNoTestDependencies(t *testing.T) {
	runCtx := &runcontext.RunContext{
		Cfg: &latest.Pipeline{},
	}

	deps, err := NewTester(runCtx).TestDependencies()

	testutil.CheckErrorAndDeepEqual(t, false, err, 0, len(deps))
}

func TestTestDependencies(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("tests/test1.yaml", "")
	tmpDir.Write("tests/test2.yaml", "")
	tmpDir.Write("test3.yaml", "")

	runCtx := &runcontext.RunContext{
		WorkingDir: tmpDir.Root(),
		Cfg: &latest.Pipeline{
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
