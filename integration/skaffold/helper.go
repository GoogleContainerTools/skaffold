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
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// Debug runs `skaffold debug` with the given arguments.
func Debug(args ...string) *RunBuilder {
	return &RunBuilder{command: "debug", args: args}
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

// Diagnose runs `skaffold diagnose` with the given arguments.
func Diagnose(args ...string) *RunBuilder {
	return &RunBuilder{command: "diagnose", args: args}
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
	logrus.Infoln(cmd.Args)

	start := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("skaffold %s: %v", b.command, err)
	}

	go func() {
		cmd.Wait()
		logrus.Infoln("Ran in", time.Since(start))
	}()

	return func() {
		cancel()
		cmd.Wait()
	}
}

// RunOrFail runs the skaffold command and fails the test
// if the command returns an error.
func (b *RunBuilder) RunOrFail(t *testing.T) {
	t.Helper()
	if err := b.Run(t); err != nil {
		t.Fatal(err)
	}
}

// Run runs the skaffold command.
func (b *RunBuilder) Run(t *testing.T) error {
	t.Helper()

	cmd := b.cmd(context.Background())
	logrus.Infoln(cmd.Args)

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "skaffold %s", b.command)
	}

	logrus.Infoln("Ran in", time.Since(start))
	return nil
}

// RunOrFailOutput runs the skaffold command and fails the test
// if the command returns an error.
// It only returns the standard output.
func (b *RunBuilder) RunOrFailOutput(t *testing.T) []byte {
	t.Helper()

	cmd := b.cmd(context.Background())
	cmd.Stdout, cmd.Stderr = nil, nil
	logrus.Infoln(cmd.Args)

	start := time.Now()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("skaffold %s: %v, %s", b.command, err, out)
	}

	logrus.Infoln("Ran in", time.Since(start))
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
	cmd.Env = append(removeSkaffoldEnvVariables(util.OSEnviron()), b.env...)
	if b.stdin != nil {
		cmd.Stdin = bytes.NewReader(b.stdin)
	}
	if b.dir != "" {
		cmd.Dir = b.dir
	}

	// If the test is killed by a timeout, go test will wait for
	// os.Stderr and os.Stdout to close as a result.
	//
	// However, the `cmd` will still run in the background
	// and hold those descriptors open.
	// As a result, go test will hang forever.
	//
	// Avoid that by wrapping stderr and stdout, breaking the short
	// circuit and forcing cmd.Run to use another pipe and goroutine
	// to pass along stderr and stdout.
	// See https://github.com/golang/go/issues/23019
	cmd.Stdout = struct{ io.Writer }{os.Stdout}
	cmd.Stderr = struct{ io.Writer }{os.Stderr}

	return cmd
}

// removeSkaffoldEnvVariables makes sure Skaffold runs without
// any env variable that might change its behaviour, such as
// enabling caching.
func removeSkaffoldEnvVariables(env []string) []string {
	var clean []string

	for _, value := range env {
		if !strings.HasPrefix(value, "SKAFFOLD_") {
			clean = append(clean, value)
		}
	}

	// Disable update check
	return append(clean, "SKAFFOLD_UPDATE_CHECK=false")
}
