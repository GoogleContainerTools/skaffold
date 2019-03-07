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

package skaffold

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
)

// RunBuilder is used to build a command line to run `skaffold`.
type RunBuilder struct {
	command    string
	configFile string
	dir        string
	ns         string
	args       []string
	env        []string
	stdin      []byte
}

// Dev runs `skaffold dev` with the given arguments.
func Dev(args ...string) *RunBuilder {
	return &RunBuilder{command: "dev", args: args}
}

// Fix runs `skaffold fix` with the given arguments.
func Fix(args ...string) *RunBuilder {
	return &RunBuilder{command: "fix", args: args}
}

// Build runs `skaffold build` with the given arguments.
func Build(args ...string) *RunBuilder {
	return &RunBuilder{command: "build", args: args}
}

// Deploy runs `skaffold deploy` with the given arguments.
func Deploy(args ...string) *RunBuilder {
	return &RunBuilder{command: "deploy", args: args}
}

// Run runs `skaffold run` with the given arguments.
func Run(args ...string) *RunBuilder {
	return &RunBuilder{command: "run", args: args}
}

// Delete runs `skaffold delete` with the given arguments.
func Delete(args ...string) *RunBuilder {
	return &RunBuilder{command: "delete", args: args}
}

// Config runs `skaffold config` with the given arguments.
func Config(args ...string) *RunBuilder {
	return &RunBuilder{command: "config", args: args}
}

// Init runs `skaffold init` with the given arguments.
func Init(args ...string) *RunBuilder {
	return &RunBuilder{command: "init", args: args}
}

// InDir sets the directory in which skaffold is running.
func (b *RunBuilder) InDir(dir string) *RunBuilder {
	b.dir = dir
	return b
}

// WithConfig sets the config file to be used by skaffold.
func (b *RunBuilder) WithConfig(configFile string) *RunBuilder {
	b.configFile = configFile
	return b
}

// WithStdin sets the stdin.
func (b *RunBuilder) WithStdin(input []byte) *RunBuilder {
	b.stdin = input
	return b
}

// InNs sets the Kubernetes namespace in which skaffold deploys.
func (b *RunBuilder) InNs(ns string) *RunBuilder {
	b.ns = ns
	return b
}

// WithEnv sets environment variables.
func (b *RunBuilder) WithEnv(env []string) *RunBuilder {
	b.env = env
	return b
}

// RunBackground runs the skaffold command in the background.
// This also returns a teardown function that stops skaffold.
func (b *RunBuilder) RunBackground(t *testing.T) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	cmd := b.cmd(ctx)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("skaffold %s: %v", b.command, err)
	}

	return func() {
		cancel()
		cmd.Wait()
	}
}

// Run runs the skaffold command and returns its output.
func (b *RunBuilder) Run(t *testing.T) ([]byte, error) {
	t.Helper()
	return b.cmd(context.Background()).Output()
}

// RunOrFail runs the skaffold command and fails the test
// if the command returns an error.
func (b *RunBuilder) RunOrFail(t *testing.T) []byte {
	t.Helper()
	out, err := b.Run(t)
	if err != nil {
		t.Fatalf("skaffold %s: %v, %s", b.command, err, out)
	}
	return out
}

func (b *RunBuilder) cmd(ctx context.Context) *exec.Cmd {
	args := []string{b.command}
	if b.ns != "" {
		args = append(args, "--namespace", b.ns)
	}
	if b.configFile != "" {
		args = append(args, "-f", b.configFile)
	}
	args = append(args, b.args...)

	cmd := exec.CommandContext(ctx, "skaffold", args...)
	cmd.Env = append(os.Environ(), b.env...)
	if b.stdin != nil {
		cmd.Stdin = bytes.NewReader(b.stdin)
	}
	if b.dir != "" {
		cmd.Dir = b.dir
	}

	return cmd
}
