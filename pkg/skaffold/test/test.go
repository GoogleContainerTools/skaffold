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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"

	"github.com/pkg/errors"
)

// Test is the top level testing execution call.
// it should return back the list of artifacts that passed the tests
// in some form, so they can be deployed. It should also be responsible for
// logging which artifacts were not deployed because their tests failed.
func (t FullTester) Test(out io.Writer, bRes []build.Artifact) error {
	t.resolveArtifactImageTags(bRes)
	for _, aTester := range t.ArtifactTesters {
		if err := aTester.RunTests(); err != nil {
			return err
		}
	}
	return nil
}

// NewTester parses the provided test cases from the Skaffold config,
// and returns a Tester instance with all the necessary test runners
// to run all specified tests.
func NewTester(testCases *[]v1alpha3.TestCase) (Tester, error) {
	testers := []*ArtifactTester{}
	deps := []string{}
	for _, testCase := range *testCases {
		testRunner := &ArtifactTester{
			ImageName: testCase.ImageName,
		}
		if testCase.StructureTests != nil {
			stRunner, err := structure.NewStructureTestRunner(testCase.StructureTests)
			if err != nil {
				return FullTester{}, errors.Wrap(err, "retrieving structure test runner")
			}
			testRunner.TestRunners = append(testRunner.TestRunners, stRunner)
			deps = append(deps, testCase.StructureTests...)
		}
		testers = append(testers, testRunner)
	}
	return FullTester{
		ArtifactTesters: testers,
		Dependencies:    deps,
	}, nil
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
