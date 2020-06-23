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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// NewTester parses the provided test cases from the Skaffold config,
// and returns a Tester instance with all the necessary test runners
// to run all specified tests.
func NewTester(runCtx *runcontext.RunContext, imagesAreLocal bool) Tester {
	localDaemon, err := docker.NewAPIClient(runCtx)
	if err != nil {
		return nil
	}

	return FullTester{
		testCases:      runCtx.Cfg.Test,
		workingDir:     runCtx.WorkingDir,
		localDaemon:    localDaemon,
		imagesAreLocal: imagesAreLocal,
	}
}

// TestDependencies returns the watch dependencies to the runner.
func (t FullTester) TestDependencies() ([]string, error) {
	var deps []string

	for _, test := range t.testCases {
		files, err := util.ExpandPathsGlob(t.workingDir, test.StructureTests)
		if err != nil {
			return nil, fmt.Errorf("expanding test file paths: %w", err)
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
			return fmt.Errorf("running structure tests: %w", err)
		}
	}

	return nil
}

func (t FullTester) runStructureTests(ctx context.Context, out io.Writer, bRes []build.Artifact, tc *latest.TestCase) error {
	if len(tc.StructureTests) == 0 {
		return nil
	}

	fqn, found := resolveArtifactImageTag(tc.ImageName, bRes)
	if !found {
		logrus.Debugln("Skipping tests for", tc.ImageName, "since it wasn't built")
		return nil
	}

	if !t.imagesAreLocal {
		// The image is remote so we have to pull it locally.
		// `container-structure-test` currently can't do it:
		// https://github.com/GoogleContainerTools/container-structure-test/issues/253.
		if err := t.localDaemon.Pull(ctx, out, fqn); err != nil {
			return fmt.Errorf("unable to docker pull image %q: %w", fqn, err)
		}
	}

	files, err := util.ExpandPathsGlob(t.workingDir, tc.StructureTests)
	if err != nil {
		return fmt.Errorf("expanding test file paths: %w", err)
	}

	runner := structure.NewRunner(files, t.localDaemon.ExtraEnv())

	return runner.Test(ctx, out, fqn)
}

func resolveArtifactImageTag(imageName string, bRes []build.Artifact) (string, bool) {
	for _, res := range bRes {
		if imageName == res.ImageName {
			return res.Tag, true
		}
	}

	return "", false
}
