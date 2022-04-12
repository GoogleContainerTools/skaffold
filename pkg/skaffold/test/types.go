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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Tester is the top level test executor in Skaffold.
// A tester is really a collection of artifact-specific testers,
// each of which contains one or more Tester which implements
// a single test run.
type Tester interface {
	Test(context.Context, io.Writer, []graph.Artifact) error
	TestDependencies(ctx context.Context, artifact *latest.Artifact) ([]string, error)
}

type Muted interface {
	MuteTest() bool
}

// FullTester serves as a holder for the individual artifact-specific
// testers. It exists so that the Tester interface can mimic the Builder/Deployer
// interface, so it can be called in a similar fashion from the runner, while
// the FullTester actually handles the work.

// FullTester should always be the ONLY implementation of the Tester interface;
// newly added testing implementations should implement the imageTester interface.
type FullTester struct {
	Testers ImageTesters
	muted   Muted
	// imagesAreLocal func(imageName string) (bool, error)
}

// ImageTester is the lowest-level test executor in Skaffold, responsible for
// running a single test on a single artifact image and returning its result.
// Any new test type should implement this interface.
type ImageTester interface {
	Test(ctx context.Context, out io.Writer, tag string) error

	TestDependencies(ctx context.Context) ([]string, error)
}

// ImageTesters is a collection of imageTester interfaces grouped by the target image name
type ImageTesters map[string][]ImageTester
