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

package structure

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Test is the entrypoint for running structure tests
func (tr *Runner) Test(ctx context.Context, out io.Writer, image string) error {
	logrus.Infof("Running structure tests for files %v", tr.testFiles)

	args := []string{"test", "-v", "warn", "--image", image}
	for _, f := range tr.testFiles {
		args = append(args, "--config", f)
	}

	cmd := exec.CommandContext(ctx, "container-structure-test", args...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = tr.env()

	if err := util.RunCmd(cmd); err != nil {
		return fmt.Errorf("running container-structure-test: %w", err)
	}

	return nil
}

// env returns a merged environment of the current process environment and any extra environment.
// This ensures that the correct docker environment configuration is passed to container-structure-test,
// for example when running on minikube.
func (tr *Runner) env() []string {
	if tr.extraEnv == nil {
		return nil
	}

	parentEnv := os.Environ()
	mergedEnv := make([]string, len(parentEnv), len(parentEnv)+len(tr.extraEnv))
	copy(mergedEnv, parentEnv)
	return append(mergedEnv, tr.extraEnv...)
}
