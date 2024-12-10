/*
Copyright 2024 The Skaffold Authors

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

package tofu

import (
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// CLI holds parameters to run OpenTofu.
type CLI struct {
	chDir string

	version     ClientVersion
	versionOnce sync.Once
}

type Config interface {
	GetWorkspace() string
}

// NewCLI creates a new tofu CLI
func NewCLI(cfg Config) *CLI {
	return &CLI{
		chDir: cfg.GetWorkspace(),
	}
}

// Command creates the underlying exec.CommandContext. This allows low-level control of the executed command.
func (c *CLI) Command(ctx context.Context, command string, arg ...string) *exec.Cmd {
	args := c.args(command, arg...)
	return exec.CommandContext(ctx, "tofu", args...)
}

// Run shells out tofu CLI.
func (c *CLI) Run(ctx context.Context, in io.Reader, out io.Writer, command string, arg ...string) error {
	cmd := c.Command(ctx, command, arg...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = append(cmd.Env, "TF_IN_AUTOMATION=true")
	return util.RunCmd(ctx, cmd)
}

// RunOut shells out tofu CLI.
func (c *CLI) RunOut(ctx context.Context, command string, arg ...string) ([]byte, error) {
	cmd := c.Command(ctx, command, arg...)
	return util.RunCmdOut(ctx, cmd)
}

// RunOutInput shells out tofu CLI with a given input stream.
func (c *CLI) RunOutInput(ctx context.Context, in io.Reader, command string, arg ...string) ([]byte, error) {
	cmd := c.Command(ctx, command, arg...)
	cmd.Stdin = in
	return util.RunCmdOut(ctx, cmd)
}

// CommandWithStrictCancellation ensures for windows OS that all child process get terminated on cancellation
func (c *CLI) CommandWithStrictCancellation(ctx context.Context, command string, arg ...string) *Cmd {
	args := c.args(command, arg...)
	return CommandContext(ctx, "tofu", args...)
}

// args builds an argument list for calling tofu
func (c *CLI) args(command string, arg ...string) []string {
	args := []string{}
	if c.chDir != "" {
		args = append(args, "-chdir="+c.chDir)
	}
	args = append(args, command)
	args = append(args, arg...)
	return args
}
