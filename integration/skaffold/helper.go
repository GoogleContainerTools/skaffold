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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// RunBuilder is used to build a command line to run `skaffold`.
type RunBuilder struct {
	command    string
	configFile string
	dir        string
	ns         string
	repo       string
	profiles   []string
	args       []string
	env        []string
	stdin      []byte
}

// Dev runs `skaffold dev` with the given arguments.
func Dev(args ...string) *RunBuilder {
	return withDefaults("dev", args)
}

// Fix runs `skaffold fix` with the given arguments.
func Fix(args ...string) *RunBuilder {
	return withDefaults("fix", args)
}

// Build runs `skaffold build` with the given arguments.
func Build(args ...string) *RunBuilder {
	return withDefaults("build", args)
}

// Deploy runs `skaffold deploy` with the given arguments.
func Deploy(args ...string) *RunBuilder {
	return withDefaults("deploy", args)
}

// Debug runs `skaffold debug` with the given arguments.
func Debug(args ...string) *RunBuilder {
	return withDefaults("debug", args)
}

// Run runs `skaffold run` with the given arguments.
func Run(args ...string) *RunBuilder {
	return withDefaults("run", args)
}

// Delete runs `skaffold delete` with the given arguments.
func Delete(args ...string) *RunBuilder {
	return withDefaults("delete", args)
}

// Config runs `skaffold config` with the given arguments.
func Config(args ...string) *RunBuilder {
	return withDefaults("config", args)
}

// Init runs `skaffold init` with the given arguments.
func Init(args ...string) *RunBuilder {
	return withDefaults("init", args)
}

// Diagnose runs `skaffold diagnose` with the given arguments.
func Diagnose(args ...string) *RunBuilder {
	return withDefaults("diagnose", args)
}

// Schema runs `skaffold schema` with the given arguments.
func Schema(args ...string) *RunBuilder {
	return &RunBuilder{command: "schema", args: args}
}

// Credits runs `skaffold credits` with the given arguments.
func Credits(args ...string) *RunBuilder {
	return &RunBuilder{command: "credits", args: args}
}

// Render runs `skaffold render` with the given arguments.
func Render(args ...string) *RunBuilder {
	return withDefaults("render", args)
}

func GeneratePipeline(args ...string) *RunBuilder {
	return withDefaults("generate-pipeline", args)
}

func withDefaults(command string, args []string) *RunBuilder {
	return &RunBuilder{command: command, args: args, repo: "gcr.io/k8s-skaffold"}
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

// WithRepo sets the default repository to be used by skaffold.
func (b *RunBuilder) WithRepo(repo string) *RunBuilder {
	b.repo = repo
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

// WithProfiles sets profiles.
func (b *RunBuilder) WithProfiles(profiles []string) *RunBuilder {
	b.profiles = profiles
	return b
}

// RunBackground runs the skaffold command in the background.
func (b *RunBuilder) RunBackground(t *testing.T) io.ReadCloser {
	t.Helper()

	pr, pw := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())

	cmd := b.cmd(ctx)
	cmd.Stdout = pw
	logrus.Infoln(cmd.Args)

	start := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("skaffold %s: %v", b.command, err)
	}

	go func() {
		cmd.Wait()
		logrus.Infoln("Ran in", time.Since(start))
	}()

	t.Cleanup(func() {
		cancel()
		cmd.Wait()
		pr.Close()
	})

	return pr
}

// RunOrFail runs the skaffold command and fails the test
// if the command returns an error.
func (b *RunBuilder) RunOrFail(t *testing.T) {
	b.RunOrFailOutput(t)
}

// Run runs the skaffold command.
func (b *RunBuilder) Run(t *testing.T) error {
	t.Helper()

	cmd := b.cmd(context.Background())
	logrus.Infoln(cmd.Args)

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("skaffold %q: %w", b.command, err)
	}

	logrus.Infoln("Ran in", time.Since(start))
	return nil
}

// RunWithCombinedOutput runs the skaffold command and returns the combined standard output and error.
func (b *RunBuilder) RunWithCombinedOutput(t *testing.T) ([]byte, error) {
	t.Helper()

	cmd := b.cmd(context.Background())
	cmd.Stdout, cmd.Stderr = nil, nil
	logrus.Infoln(cmd.Args)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("skaffold %q: %w", b.command, err)
	}

	logrus.Infoln("Ran in", time.Since(start))
	return out, nil
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
		if ee, ok := err.(*exec.ExitError); ok {
			defer t.Errorf(string(ee.Stderr))
		}
		t.Fatalf("skaffold %s: %v, %s", b.command, err, out)
	}

	logrus.Infoln("Ran in", time.Since(start))
	return out
}

func (b *RunBuilder) cmd(ctx context.Context) *exec.Cmd {
	args := []string{b.command}
	command := b.getCobraCommand()

	if b.ns != "" && command.Flags().Lookup("namespace") != nil {
		args = append(args, "--namespace", b.ns)
	}
	if b.configFile != "" && command.Flags().ShorthandLookup("f") != nil {
		args = append(args, "-f", b.configFile)
	}
	if b.repo != "" && command.Flags().Lookup("default-repo") != nil {
		args = append(args, "--default-repo", b.repo)
	}
	if len(b.profiles) > 0 && command.Flags().Lookup("profile") != nil {
		args = append(args, "--profile", strings.Join(b.profiles, ","))
	}
	args = append(args, b.args...)

	skaffoldBinary := "skaffold"
	if value, found := os.LookupEnv("SKAFFOLD_BINARY"); found {
		skaffoldBinary = value
	}
	cmd := exec.CommandContext(ctx, skaffoldBinary, args...)
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

// getCobraCommand returns the matching cobra command for the command
// in b, or return a dummy without flags
func (b *RunBuilder) getCobraCommand() *cobra.Command {
	c := cmd.NewSkaffoldCommand(os.Stdout, os.Stderr)
	for _, comm := range c.Commands() {
		if comm.Name() == b.command {
			return comm
		}
	}
	return &cobra.Command{}
}

// removeSkaffoldEnvVariables makes sure Skaffold runs without
// any env variable that might change its behavior, such as
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
