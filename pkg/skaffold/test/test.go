/*
Copyright 2018 The Skaffold Authors

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
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/pkg/errors"
)

// NewTester parses the provided test cases from the Skaffold config,
// and returns a Tester instance with all the necessary test runners
// to run all specified tests.
func NewTester(testCases *[]latest.TestCase) (Tester, error) {
	testers := []*ArtifactTester{}
	deps := []string{}
	// TODO(nkubala): copied this from runner.getDeployer(), this should be moved somewhere else
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "finding current directory")
	}
	for _, testCase := range *testCases {
		testRunner := &ArtifactTester{
			ImageName: testCase.ImageName,
		}
		if testCase.StructureTests != nil {
			stFiles, err := util.ExpandPathsGlob(cwd, testCase.StructureTests)
			if err != nil {
				return FullTester{}, errors.Wrap(err, "expanding test file paths")
			}
			stRunner, err := structure.NewStructureTestRunner(stFiles)
			if err != nil {
				return FullTester{}, errors.Wrap(err, "retrieving structure test runner")
			}
			testRunner.TestRunners = append(testRunner.TestRunners, stRunner)

			deps = append(deps, stFiles...)
		}
		testers = append(testers, testRunner)
	}
	return FullTester{
		ArtifactTesters: testers,
		Dependencies:    deps,
	}, nil
}

// TestDependencies returns the watch dependencies to the runner.
func (t FullTester) TestDependencies() []string {
	return t.Dependencies
}

// Test is the top level testing execution call. It serves as the
// entrypoint to all individual tests.
func (t FullTester) Test(out io.Writer, bRes []build.Artifact) error {
	t.resolveArtifactImageTags(bRes)
	for _, aTester := range t.ArtifactTesters {
		if err := aTester.RunTests(); err != nil {
			return err
		}
	}
	return nil
}

// RunTests serves as the entrypoint to each group of
// artifact-specific tests.
func (a *ArtifactTester) RunTests() error {
	for _, t := range a.TestRunners {
		if err := t.Test(a.ImageName); err != nil {
			return err
		}
	}
	return nil
}

// replace original test artifact images with tagged build artifact images
func (t *FullTester) resolveArtifactImageTags(bRes []build.Artifact) {
	for _, aTest := range t.ArtifactTesters {
		for _, res := range bRes {
			if aTest.ImageName == res.ImageName {
				aTest.ImageName = res.Tag
			}
		}
	}
}
