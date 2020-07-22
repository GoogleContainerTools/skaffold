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

package kubectl

import (
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// CLI holds parameters to run kubectl.
type CLI struct {
	KubeContext string
	KubeConfig  string
	Namespace   string

	version     ClientVersion
	versionOnce sync.Once
}

func NewFromRunContext(runCtx *runcontext.RunContext) *CLI {
	return &CLI{
		KubeContext: runCtx.KubeContext,
		KubeConfig:  runCtx.Opts.KubeConfig,
		Namespace:   runCtx.Opts.Namespace,
	}
}

// Command creates the underlying exec.CommandContext. This allows low-level control of the executed command.
func (c *CLI) Command(ctx context.Context, command string, arg ...string) *exec.Cmd {
	args := c.args(command, "", arg...)
	return exec.CommandContext(ctx, "kubectl", args...)
}

// Command creates the underlying exec.CommandContext with namespace. This allows low-level control of the executed command.
func (c *CLI) CommandWithNamespaceArg(ctx context.Context, command string, namespace string, arg ...string) *exec.Cmd {
	args := c.args(command, namespace, arg...)
	return exec.CommandContext(ctx, "kubectl", args...)
}

// Run shells out kubectl CLI.
func (c *CLI) Run(ctx context.Context, in io.Reader, out io.Writer, command string, arg ...string) error {
	cmd := c.Command(ctx, command, arg...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	return util.RunCmd(cmd)
}

// RunInNamespace shells out kubectl CLI with given namespace
func (c *CLI) RunInNamespace(ctx context.Context, in io.Reader, out io.Writer, command string, namespace string, arg ...string) error {
	cmd := c.CommandWithNamespaceArg(ctx, command, namespace, arg...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	return util.RunCmd(cmd)
}

// RunOut shells out kubectl CLI.
func (c *CLI) RunOut(ctx context.Context, command string, arg ...string) ([]byte, error) {
	cmd := c.Command(ctx, command, arg...)
	return util.RunCmdOut(cmd)
}

// RunOutInput shells out kubectl CLI with a given input stream.
func (c *CLI) RunOutInput(ctx context.Context, in io.Reader, command string, arg ...string) ([]byte, error) {
	cmd := c.Command(ctx, command, arg...)
	cmd.Stdin = in
	return util.RunCmdOut(cmd)
}

// CommandWithStrictCancellation ensures for windows OS that all child process get terminated on cancellation
func (c *CLI) CommandWithStrictCancellation(ctx context.Context, command string, arg ...string) *Cmd {
	args := c.args(command, "", arg...)
	return CommandContext(ctx, "kubectl", args...)
}

// args builds an argument list for calling kubectl and consistently
// adds the `--context` and `--namespace` flags.
func (c *CLI) args(command string, namespace string, arg ...string) []string {
	args := []string{"--context", c.KubeContext}
	namespace = c.resolveNamespace(namespace)
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	if c.KubeConfig != "" {
		args = append(args, "--kubeconfig", c.KubeConfig)
	}
	args = append(args, command)
	args = append(args, arg...)
	return args
}

func (c *CLI) resolveNamespace(ns string) string {
	if ns != "" {
		return ns
	}
	return c.Namespace
}
