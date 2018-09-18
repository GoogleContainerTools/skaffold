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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"io"
)

// Tester is the top level test executor in Skaffold.
// A tester is really a collection of artifact-specific testers,
// each of which contains one or more TestRunners which implements
// a single test run.
type Tester interface {
	Test(io.Writer, []build.Artifact) error

	TestDependencies() []string
}

type FullTester struct {
	ArtifactTesters []*ArtifactTester
	Dependencies    []string
}

func (t FullTester) TestDependencies() []string {
	return t.Dependencies
}

// ArtifactTester is an artifact-specific test holder, which contains
// tests runners to run all specified tests on an individual artifact.
type ArtifactTester struct {
	ImageName   string
	TestRunners []TestRunner
}

func (a *ArtifactTester) RunTests() error {
	for _, t := range a.TestRunners {
		if err := t.Test(a.ImageName); err != nil {
			return err
		}
	}
	return nil
}

// TestRunner is the lowest-level test executor in Skaffold, responsible for
// running a single test on a single artifact image and returning its result.
type TestRunner interface {
	Test(image string) error
}
