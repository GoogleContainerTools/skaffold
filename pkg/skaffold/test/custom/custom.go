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
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/list"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type Runner struct {
	customTest     latest.CustomTest
	image          string
	testWorkingDir string
	localDaemon    docker.LocalDaemon
}

// New creates a new custom.Runner.
func New(cfg docker.Config, wd string, ct latest.CustomTest) (*Runner, error) {
	localDaemon, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Runner{
		customTest:     ct,
		testWorkingDir: wd,
		localDaemon:    localDaemon,
	}, nil
}

// Test is the entrypoint for running custom tests
func (ct *Runner) Test(ctx context.Context, out io.Writer, _ []build.Artifact) error {
	if msg, err := ct.runCustomCommand(ctx, out); msg == "" {
		return cutomTestErr(err)
	}

	return nil
}

func (ct *Runner) runCustomCommand(ctx context.Context, out io.Writer) (string, error) {
	test := ct.customTest

	// Expand command
	command, err := util.ExpandEnvTemplate(test.Command, nil)
	if err != nil {
		return "", parsingTestCommandErr(test.Command, err)
	}

	if len(test.TimeoutSeconds) != 0 {
		// Create a new context wiht timeout
		timeout, err := strconv.Atoi(test.TimeoutSeconds)
		if err != nil {
			return "", retrievingTimeoutErr(err)
		}

		newCtx, cancel := context.WithTimeout(ctx, (time.Duration(timeout))*(time.Second))
		defer cancel()
		ctx = newCtx
	}

	var cmd *exec.Cmd
	// We evaluate the command with a shell so that it can contain
	// env variables.
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = ct.env()

	if err := cmd.Run(); err != nil {
		return "", runCmdErr(err)
	}

	// check the context error to see if the timeout was executed.
	if ctx.Err() == context.DeadlineExceeded {
		return "", commandExecutionTimedoutErr(err)
	}

	// If there's no context error, we know the command completed (or errored).
	if err != nil {
		return "", commandNonZeroExitErr(err)
	}

	return "", misc.HandleGracefulTermination(ctx, cmd)
}

// env returns a merged environment of the current process environment and any extra environment.
func (ct *Runner) env() []string {
	extraEnv := ct.localDaemon.ExtraEnv()
	if extraEnv == nil {
		return nil
	}

	parentEnv := os.Environ()
	mergedEnv := make([]string, len(parentEnv), len(parentEnv)+len(extraEnv))
	copy(mergedEnv, parentEnv)
	return append(mergedEnv, extraEnv...)
}

// TestDependencies returns dependencies listed for a custom test
func (ct *Runner) TestDependencies() ([]string, error) {
	test := ct.customTest

	switch {
	case test.Dependencies.Command != "":
		split := strings.Split(test.Dependencies.Command, " ")
		cmd := exec.CommandContext(context.Background(), split[0], split[1:]...)
		output, err := util.RunCmdOut(cmd)
		if err != nil {
			return nil, gettingDependenciesCommandErr(test.Dependencies.Command, err)
		}
		var deps []string
		if err := json.Unmarshal(output, &deps); err != nil {
			return nil, dependencyOutputUnmarshallErr(err)
		}
		return deps, nil

	default:
		return list.Files(ct.testWorkingDir, test.Dependencies.Paths, test.Dependencies.Ignore)
	}
}
