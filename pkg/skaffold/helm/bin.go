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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver"
	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

var (
	// VersionRegex extracts version from "helm version", for instance: "v3.19.0"
	VersionRegex = regexp.MustCompile(`v?(\d[\w.\-]+)`)

	// OSExecutable allows for replacing the skaffold binary for testing purposes
	OSExecutable = os.Executable

	// PluginInstallDir allows for replacing the `helm plugin install ...` directory for testing purposes
	PluginInstallDir = ""

	// WriteBuildArtifacts is meant to be reassigned for testing
	WriteBuildArtifacts = writeBuildArtifacts
)

// Helm4PostRendererVersion represents the version cut-off for Helm v4's
// introduction of post-renderer plugins, not supporting
// `--post-renderer <executable>` anymore
var Helm4PostRendererVersion = semver.MustParse("4.0.0-beta.1")

const PostRendererTemplate = `
name: "%s"
version: "0.1"
type: postrenderer/v1
apiVersion: v1
runtime: subprocess
runtimeConfig:
  platformCommand:
    - command: %s
`

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
	// Helm v2 needed the `--client` to avoid connecting to Kubernetes.
	// Support for Helm v2 was dropped here.
	cmd := exec.CommandContext(ctx, "helm", "version")
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

// PreparePostRenderer conditionally installs a post renderer plugin (starting from Helm v4) and returns
// an optional cleanup function in that case, plus in any case the needed command line arguments
func PreparePostRenderer(ctx context.Context, h Client, skaffoldBinary string, helmVersion semver.Version) (func(), []string, error) {
	if helmVersion.LT(Helm4PostRendererVersion) {
		return nil, []string{"--post-renderer", skaffoldBinary}, nil
	}

	// Helm v4 logic

	skaffoldBinaryAsYaml, err := yaml.Marshal(skaffoldBinary)
	if err != nil {
		return nil, nil, PluginErr("Failed to fill Skaffold executable into Helm plugin manifest", err)
	}

	var helmPluginDir string
	if PluginInstallDir == "" {
		helmPluginDir, err = os.MkdirTemp("", "skaffold-render")
		if err != nil {
			return nil, nil, PluginErr("Failed to create temporary directory for Helm plugin manifest", err)
		}
	} else {
		helmPluginDir = PluginInstallDir
	}

	// Take the temporary directory name as unique plugin name. That string is expected to
	// be safe for direct %s insertion in the YAML Helm plugin manifest.
	helmPluginName := path.Base(helmPluginDir)
	err = os.WriteFile(path.Join(helmPluginDir, "plugin.yaml"), []byte(fmt.Sprintf(PostRendererTemplate, helmPluginName, skaffoldBinaryAsYaml)), 0644)
	if err != nil &&
		// Don't produce an error in tests that simulate the directory
		!strings.HasPrefix(PluginInstallDir, "TEMPORARY-TEST-DIR") {
		os.RemoveAll(helmPluginDir)
		return nil, nil, PluginErr("Failed to write Helm plugin.yaml", err)
	}

	out := new(bytes.Buffer)
	err = Exec(ctx, h, out, false, nil, "plugin", "install", helmPluginDir)
	if err != nil {
		os.RemoveAll(helmPluginDir)
		return nil, nil, PluginErr("Failed to install Helm plugin", errors.New(strings.TrimSpace(out.String())))
	}

	cleanUp := func() {
		Exec(ctx, h, out, false, nil, "plugin", "uninstall", helmPluginName)
		os.RemoveAll(helmPluginDir)
	}
	return cleanUp, []string{"--post-renderer", helmPluginName}, nil
}

func PrepareSkaffoldFilter(h Client, builds []graph.Artifact, flags []string) (skaffoldBinary string, env []string, cleanup func(), err error) {
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
	cmdLine := generateSkaffoldFilter(h, buildsFile, flags)
	env = append(env, fmt.Sprintf("SKAFFOLD_CMDLINE=%s", shell.Join(cmdLine...)))
	env = append(env, fmt.Sprintf("SKAFFOLD_FILENAME=%s", h.ConfigFile()))
	return
}

// generateSkaffoldFilter creates a "skaffold filter" command-line for applying the various
// Skaffold manifest filters, such a debugging, image replacement, and applying labels.
func generateSkaffoldFilter(h Client, buildsFile string, flags []string) []string {
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

	if h.KubeConfig() != "" {
		args = append(args, "--kubeconfig", h.KubeConfig())
	}

	args = append(args, flags...)
	return args
}

func generateHelmCommand(ctx context.Context, h Client, useSecrets bool, env []string, args ...string) *exec.Cmd {
	// Only add Kubernetes parameters for subcommands that need it. The plugin system
	// shouldn't need it.
	wantKubernetesArgs := h != nil && (len(args) == 0 || args[0] != "plugin")

	if wantKubernetesArgs {
		args = append([]string{"--kube-context", h.KubeContext()}, args...)
	}
	args = append(args, h.GlobalFlags()...)

	if wantKubernetesArgs && h.KubeConfig() != "" {
		args = append(args, "--kubeconfig", h.KubeConfig())
	}

	if useSecrets {
		args = append([]string{"secrets"}, args...)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Cancel = func() error {
		fmt.Println("Terminating helm, giving it 2 minutes to clean up...")
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 120 * time.Second

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
