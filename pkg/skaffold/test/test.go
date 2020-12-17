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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/logfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type Config interface {
	docker.Config

	TestCases() []*latest.TestCase
	GetWorkingDir() string
	Muted() config.Muted
}

// NewTester parses the provided test cases from the Skaffold config,
// and returns a Tester instance with all the necessary test runners
// to run all specified tests.
func NewTester(cfg Config, imagesAreLocal func(imageName string) (bool, error)) Tester {
	localDaemon, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil
	}

	return FullTester{
		testCases:      cfg.TestCases(),
		workingDir:     cfg.GetWorkingDir(),
		muted:          cfg.Muted(),
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
	if len(t.testCases) == 0 {
		return nil
	}

	color.Default.Fprintln(out, "Testing images...")

	if t.muted.MuteTest() {
		file, err := logfile.Create("test.log")
		if err != nil {
			return fmt.Errorf("unable to create log file for tests: %w", err)
		}
		fmt.Fprintln(out, " - writing logs to", file.Name())

		// Print logs to a memory buffer and to a file.
		var buf bytes.Buffer
		w := io.MultiWriter(file, &buf)

		// Run the tests.
		err = t.runTests(ctx, w, bRes)

		// After the test finish, close the log file. If the tests failed, print the full log to the console.
		file.Close()
		if err != nil {
			buf.WriteTo(out)
		}

		return err
	}

	return t.runTests(ctx, out, bRes)
}

func (t FullTester) runTests(ctx context.Context, out io.Writer, bRes []build.Artifact) error {
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

	if imageIsLocal, err := t.imagesAreLocal(tc.ImageName); err != nil {
		return err
	} else if !imageIsLocal {
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
