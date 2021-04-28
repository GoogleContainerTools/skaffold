/*
Copyright 2021 The Skaffold Authors

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

package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/list"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	// for tests
	testContext = retrieveTestContext
)

const Windows string = "windows"

type Runner struct {
	customTest latest_v1.CustomTest
	imageName  string
	workspace  string
}

// New creates a new custom.Runner.
func New(cfg docker.Config, imageName string, ws string, ct latest_v1.CustomTest) (*Runner, error) {
	return &Runner{
		imageName:  imageName,
		customTest: ct,
		workspace:  ws,
	}, nil
}

// Test is the entrypoint for running custom tests
func (ct *Runner) Test(ctx context.Context, out io.Writer, imageTag string) error {
	event.TestInProgress()
	if err := ct.runCustomTest(ctx, out, imageTag); err != nil {
		event.TestFailed(ct.imageName, err)
		return err
	}
	event.TestComplete()
	return nil
}

func (ct *Runner) runCustomTest(ctx context.Context, out io.Writer, imageTag string) error {
	test := ct.customTest

	// Expand command
	command, err := util.ExpandEnvTemplate(test.Command, nil)
	if err != nil {
		return cmdRunParsingErr(test.Command, err)
	}

	if test.TimeoutSeconds <= 0 {
		color.Default.Fprintf(out, "Running custom test command: %q\n", command)
	} else {
		color.Default.Fprintf(out, "Running custom test command: %q with timeout %d s\n", command, test.TimeoutSeconds)
		newCtx, cancel := context.WithTimeout(ctx, time.Duration(test.TimeoutSeconds)*time.Second)

		defer cancel()
		ctx = newCtx
	}

	cmd, err := ct.retrieveCmd(ctx, out, command, imageTag)
	if err != nil {
		return cmdRunRetrieveErr(command, ct.imageName, err)
	}

	if err := util.RunCmd(cmd); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			// If the process exited by itself, just return the error
			if e.Exited() {
				color.Red.Fprintf(out, "Command finished with non-0 exit code.\n")
				return cmdRunNonZeroExitErr(command, e)
			}
			// If the context is done, it has been killed by the exec.Command
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					color.Red.Fprintf(out, "Command timed out\n")
					return cmdRunTimedoutErr(test.TimeoutSeconds, ctx.Err())
				} else if ctx.Err() == context.Canceled {
					color.Red.Fprintf(out, "Command cancelled\n")
					return cmdRunCancelledErr(ctx.Err())
				}
				return cmdRunExecutionErr(ctx.Err())
			default:
				return cmdRunExited(e)
			}
		}
		return cmdRunErr(err)
	}
	color.Green.Fprintf(out, "Command finished successfully.\n")

	return nil
}

// TestDependencies returns dependencies listed for a custom test
func (ct *Runner) TestDependencies() ([]string, error) {
	test := ct.customTest

	if test.Dependencies != nil {
		switch {
		case test.Dependencies.Command != "":
			var cmd *exec.Cmd
			// We evaluate the command with a shell so that it can contain env variables.
			if runtime.GOOS == Windows {
				cmd = exec.CommandContext(context.Background(), "cmd.exe", "/C", test.Dependencies.Command)
			} else {
				cmd = exec.CommandContext(context.Background(), "sh", "-c", test.Dependencies.Command)
			}

			output, err := util.RunCmdOut(cmd)
			if err != nil {
				return nil, gettingDependenciesCommandErr(test.Dependencies.Command, err)
			}
			var deps []string
			if err := json.Unmarshal(output, &deps); err != nil {
				return nil, dependencyOutputUnmarshallErr(test.Dependencies.Paths, err)
			}
			return deps, nil

		case test.Dependencies.Paths != nil:
			return list.Files(ct.workspace, test.Dependencies.Paths, test.Dependencies.Ignore)
		}
	}
	return nil, nil
}

func (ct *Runner) retrieveCmd(ctx context.Context, out io.Writer, command string, imageTag string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	// We evaluate the command with a shell so that it can contain env variables.
	if runtime.GOOS == Windows {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Stdout = out
	cmd.Stderr = out

	env, err := ct.getEnv(imageTag)
	if err != nil {
		return nil, fmt.Errorf("setting env variables: %w", err)
	}
	cmd.Env = env

	dir, err := testContext(ct.workspace)
	if err != nil {
		return nil, fmt.Errorf("getting context for test: %w", err)
	}
	cmd.Dir = dir

	return cmd, nil
}

func (ct *Runner) getEnv(imageTag string) ([]string, error) {
	testContext, err := testContext(ct.workspace)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path for test context: %w", err)
	}

	envs := []string{
		fmt.Sprintf("%s=%s", "IMAGE", imageTag),
		fmt.Sprintf("%s=%s", "TEST_CONTEXT", testContext),
	}

	envs = append(envs, util.OSEnviron()...)
	return envs, nil
}

func retrieveTestContext(workspace string) (string, error) {
	return filepath.Abs(workspace)
}
