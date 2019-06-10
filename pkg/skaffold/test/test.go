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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/pkg/errors"
)

// NewTester parses the provided test cases from the Skaffold config,
// and returns a Tester instance with all the necessary test runners
// to run all specified tests.
func NewTester(runCtx *runcontext.RunContext) Tester {
	return FullTester{
		testCases:  runCtx.Cfg.Test,
		workingDir: runCtx.WorkingDir,
	}
}

// TestDependencies returns the watch dependencies to the runner.
func (t FullTester) TestDependencies() ([]string, error) {
	var deps []string

	for _, test := range t.testCases {
		files, err := util.ExpandPathsGlob(t.workingDir, test.StructureTests)
		if err != nil {
			return nil, errors.Wrap(err, "expanding test file paths")
		}

		deps = append(deps, files...)
	}

	return deps, nil
}

// Test is the top level testing execution call. It serves as the
// entrypoint to all individual tests.
func (t FullTester) Test(ctx context.Context, out io.Writer, bRes []build.Artifact) error {
	for _, test := range t.testCases {
		if err := t.runStructureTests(ctx, out, bRes, test); err != nil {
			return errors.Wrap(err, "running structure tests")
		}
	}

	return nil
}

func (t FullTester) runStructureTests(ctx context.Context, out io.Writer, bRes []build.Artifact, testCase *latest.TestCase) error {
	if len(testCase.StructureTests) == 0 {
		return nil
	}

	files, err := util.ExpandPathsGlob(t.workingDir, testCase.StructureTests)
	if err != nil {
		return errors.Wrap(err, "expanding test file paths")
	}

	fqn := resolveArtifactImageTag(testCase.ImageName, bRes)

	runner := structure.NewRunner(files)
	return runner.Test(ctx, out, fqn)
}

func resolveArtifactImageTag(imageName string, bRes []build.Artifact) string {
	for _, res := range bRes {
		if imageName == res.ImageName {
			return res.Tag
		}
	}

	return imageName
}
