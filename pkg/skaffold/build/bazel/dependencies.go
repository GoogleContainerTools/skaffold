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

package bazel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

const sourceQuery = "kind('source file', deps('%[1]s')) union buildfiles(deps('%[1]s'))"

func query(target string) string {
	return fmt.Sprintf(sourceQuery, target)
}

var once sync.Once

var workspaceFileCandidates = []string{"WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel"}

// GetDependencies finds the sources dependencies for the given bazel artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, dir string, a *latest.BazelArtifact) ([]string, error) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	go func() {
		<-timer.C
		once.Do(func() { log.Entry(ctx).Warn("Retrieving Bazel dependencies can take a long time the first time") })
	}()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to find absolute path for %q: %w", dir, err)
	}
	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve symlinks in %q: %w", absDir, err)
	}

	workspaceDir, workspaceFiles, err := findWorkspace(ctx, absDir)
	if err != nil {
		return nil, fmt.Errorf("unable to find the WORKSPACE file: %w", err)
	}

	cmd := exec.CommandContext(ctx, "bazel", "query", query(a.BuildTarget), "--noimplicit_deps", "--order_output=no", "--output=label")
	cmd.Dir = dir
	stdout, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("getting bazel dependencies: %w", err)
	}

	labels := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range labels {
		if strings.HasPrefix(l, "@") {
			continue
		}
		if strings.HasPrefix(l, "//external") {
			continue
		}
		if l == "" {
			continue
		}

		rel, err := filepath.Rel(absDir, filepath.Join(workspaceDir, depToPath(l)))

		if err != nil {
			return nil, fmt.Errorf("unable to find absolute path: %w", err)
		}
		deps = append(deps, rel)
	}

	for _, workspaceFile := range workspaceFiles {
		rel, err := filepath.Rel(absDir, filepath.Join(workspaceDir, workspaceFile))
		if err != nil {
			return nil, fmt.Errorf("unable to find absolute path: %w", err)
		}
		deps = append(deps, rel)
	}

	log.Entry(ctx).Debugf("Found dependencies for bazel artifact: %v", deps)

	return deps, nil
}

func depToPath(dep string) string {
	return strings.TrimPrefix(strings.Replace(strings.TrimPrefix(dep, "//"), ":", "/", 1), "/")
}

func findWorkspace(ctx context.Context, workingDir string) (string, []string, error) {
	cmd := exec.CommandContext(ctx, "bazel", "info", "workspace")
	cmd.Dir = workingDir
	dirBytes, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", nil, fmt.Errorf("getting bazel workspace: %w", err)
	}
	dir := strings.TrimSpace(string(dirBytes))

	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", nil, fmt.Errorf("unable to resolve symlinks in %q: %w", dir, err)
	}

	var workspaceFiles []string

	for _, workspaceFile := range workspaceFileCandidates {
		if _, err := os.Stat(filepath.Join(resolvedDir, workspaceFile)); err == nil {
			workspaceFiles = append(workspaceFiles, workspaceFile)
		}
	}

	return resolvedDir, workspaceFiles, nil
}
