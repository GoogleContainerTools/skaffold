/*
Copyright 2022 The Skaffold Authors

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

package helm

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/blang/semver"
	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

var (
	// VersionRegex extracts version from "helm version --client", for instance: "2.14.0-rc.2"
	VersionRegex = regexp.MustCompile(`v?(\d[\w.\-]+)`)

	// OSExecutable allows for replacing the skaffold binary for testing purposes
	OSExecutable = os.Executable

	// WriteBuildArtifacts is meant to be reassigned for testing
	WriteBuildArtifacts = writeBuildArtifacts
)

type Client interface {
	EnableDebug() bool
	OverrideProtocols() []string
	ConfigFile() string
	KubeConfig() string
	KubeContext() string
	Labels() map[string]string
	GlobalFlags() []string
	ManifestOverrides() map[string]string
}

// BinVer returns the version of the helm binary found in PATH.
func BinVer(ctx context.Context) (semver.Version, error) {
	cmd := exec.Command("helm", "version", "--client")
	b, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return semver.Version{}, fmt.Errorf("helm version command failed %q: %w", string(b), err)
	}
	raw := string(b)
	matches := VersionRegex.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return semver.Version{}, fmt.Errorf("unable to parse output: %q", raw)
	}
	return semver.ParseTolerant(matches[1])
}

func PrepareSkaffoldFilter(h Client, builds []graph.Artifact) (skaffoldBinary string, env []string, cleanup func(), err error) {
	skaffoldBinary, err = OSExecutable()
	if err != nil {
		return "", nil, nil, fmt.Errorf("cannot locate this Skaffold binary: %w", err)
	}

	var buildsFile string
	if len(builds) > 0 {
		buildsFile, cleanup, err = WriteBuildArtifacts(builds)
		if err != nil {
			return "", nil, nil, fmt.Errorf("could not write build-artifacts: %w", err)
		}
	}
	cmdLine := generateSkaffoldFilter(h, buildsFile)
	env = append(env, fmt.Sprintf("SKAFFOLD_CMDLINE=%s", shell.Join(cmdLine...)))
	env = append(env, fmt.Sprintf("SKAFFOLD_FILENAME=%s", h.ConfigFile()))
	return
}

// generateSkaffoldFilter creates a "skaffold filter" command-line for applying the various
// Skaffold manifest filters, such a debugging, image replacement, and applying labels.
func generateSkaffoldFilter(h Client, buildsFile string) []string {
	args := []string{"filter", "--kube-context", h.KubeContext()}
	if h.EnableDebug() {
		args = append(args, "--debugging")
		for _, overrideProtocol := range h.OverrideProtocols() {
			args = append(args, fmt.Sprintf("--protocols=%s", overrideProtocol))
		}
	}
	for k, v := range h.Labels() {
		args = append(args, fmt.Sprintf("--label=%s=%s", k, v))
	}
	for k, v := range h.ManifestOverrides() {
		args = append(args, fmt.Sprintf("--set=%s=%s", k, v))
	}
	if len(buildsFile) > 0 {
		args = append(args, "--build-artifacts", buildsFile)
	}
	args = append(args, h.GlobalFlags()...)

	if h.KubeConfig() != "" {
		args = append(args, "--kubeconfig", h.KubeConfig())
	}
	return args
}

func generateHelmCommand(ctx context.Context, h Client, useSecrets bool, env []string, args ...string) *exec.Cmd {
	args = append([]string{"--kube-context", h.KubeContext()}, args...)
	args = append(args, h.GlobalFlags()...)

	if h.KubeConfig() != "" {
		args = append(args, "--kubeconfig", h.KubeConfig())
	}

	if useSecrets {
		args = append([]string{"secrets"}, args...)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	return cmd
}

// Exec executes the helm command, writing combined stdout/stderr to the provided writer
func Exec(ctx context.Context, h Client, out io.Writer, useSecrets bool, env []string, args ...string) error {
	cmd := generateHelmCommand(ctx, h, useSecrets, env, args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(ctx, cmd)
}

// ExecWithStdoutAndStderr executes the helm command, writing combined stdout and stderr to the provided writers
func ExecWithStdoutAndStderr(ctx context.Context, h Client, stdout io.Writer, stderr io.Writer, useSecrets bool, env []string, args ...string) error {
	cmd := generateHelmCommand(ctx, h, useSecrets, env, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return util.RunCmd(ctx, cmd)
}
