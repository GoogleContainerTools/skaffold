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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const sourceQuery = "kind('source file', deps('%[1]s')) union buildfiles(deps('%[1]s'))"

func query(target string) string {
	return fmt.Sprintf(sourceQuery, target)
}

var once sync.Once

// GetDependencies finds the sources dependencies for the given bazel artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, dir string, a *latest.BazelArtifact) ([]string, error) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	go func() {
		<-timer.C
		once.Do(func() { logrus.Warnln("Retrieving Bazel dependencies can take a long time the first time") })
	}()

	topLevelFolder, err := findWorkspace(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to find the WORKSPACE file: %w", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to find absolute path for %q: %w", dir, err)
	}

	cmd := exec.CommandContext(ctx, "bazel", "query", query(a.BuildTarget), "--noimplicit_deps", "--order_output=no", "--output=label")
	cmd.Dir = dir
	stdout, err := util.RunCmdOut(cmd)
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

		rel, err := filepath.Rel(absDir, filepath.Join(topLevelFolder, depToPath(l)))
		if err != nil {
			return nil, fmt.Errorf("unable to find absolute path: %w", err)
		}
		deps = append(deps, rel)
	}

	rel, err := filepath.Rel(absDir, filepath.Join(topLevelFolder, "WORKSPACE"))
	if err != nil {
		return nil, fmt.Errorf("unable to find absolute path: %w", err)
	}
	deps = append(deps, rel)

	logrus.Debugf("Found dependencies for bazel artifact: %v", deps)

	return deps, nil
}

func depToPath(dep string) string {
	return strings.TrimPrefix(strings.Replace(strings.TrimPrefix(dep, "//"), ":", "/", 1), "/")
}

func findWorkspace(workingDir string) (string, error) {
	dir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("invalid working dir: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "WORKSPACE")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no WORKSPACE file found")
		}
		dir = parent
	}
}
