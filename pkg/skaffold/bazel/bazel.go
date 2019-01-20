/*
Copyright 2018 The Skaffold Authors

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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const sourceQuery = "kind('source file', deps('%[1]s')) union buildfiles('%[1]s')"

func query(target string) string {
	return fmt.Sprintf(sourceQuery, target)
}

// GetDependencies finds the sources dependencies for the given bazel artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, workspace string, a *latest.BazelArtifact) ([]string, error) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	go func() {
		<-timer.C
		logrus.Warnln("Retrieving Bazel dependencies can take a long time the first time")
	}()

	cmd := exec.CommandContext(ctx, "bazel", "query", query(a.BuildTarget), "--noimplicit_deps", "--order_output=no")
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting bazel dependencies")
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

		deps = append(deps, depToPath(l))
	}

	if _, err := os.Stat(filepath.Join(workspace, "WORKSPACE")); err == nil {
		deps = append(deps, "WORKSPACE")
	}

	logrus.Debugf("Found dependencies for bazel artifact: %v", deps)

	return deps, nil
}

func depToPath(dep string) string {
	return strings.TrimPrefix(strings.Replace(strings.TrimPrefix(dep, "//"), ":", "/", 1), "/")
}
