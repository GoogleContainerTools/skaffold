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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/logfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/custom"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test/structure"
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
func NewTester(cfg Config, imagesAreLocal func(imageName string) (bool, error)) (Tester, error) {
	runner, err := getRunner(cfg, imagesAreLocal, cfg.TestCases())
	if err != nil {
		return nil, err
	}

	return FullTester{
		runners: runner,
		muted:   cfg.Muted(),
	}, nil
}

// TestDependencies returns the watch dependencies to the runner.
func (t FullTester) TestDependencies() ([]string, error) {
	var deps []string
	for _, tester := range t.runners {
		result, err := tester.TestDependencies()
		if err != nil {
			return nil, err
		}
		deps = append(deps, result...)
	}
	return deps, nil
}

// Test is the top level testing execution call. It serves as the
// entrypoint to all individual tests.
func (t FullTester) Test(ctx context.Context, out io.Writer, bRes []build.Artifact) error {
	if len(t.runners) == 0 {
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
	for _, tester := range t.runners {
		if err := tester.Test(ctx, out, bRes); err != nil {
			return fmt.Errorf("running tests: %w", err)
		}
	}
	return nil
}

func getRunner(cfg Config, imagesAreLocal func(imageName string) (bool, error), tcs []*latest.TestCase) ([]runner, error) {
	var runners []runner
	for _, tc := range tcs {
		if len(tc.StructureTests) != 0 {
			structureRunner, err := structure.New(cfg, cfg.GetWorkingDir(), tc, imagesAreLocal)
			if err != nil {
				return nil, err
			}
			runners = append(runners, structureRunner)
		}

		for _, customTest := range tc.CustomTests {
			customRunner, err := custom.New(cfg, tc.ImageName, cfg.GetWorkingDir(), customTest)
			if err != nil {
				return nil, err
			}
			runners = append(runners, customRunner)
		}
	}
	return runners, nil
}
