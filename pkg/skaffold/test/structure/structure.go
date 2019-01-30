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

package structure

import (
	"context"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	if err := util.RunCmd(cmd); err != nil {
		return errors.Wrap(err, "running container-structure-test")
	}

	return nil
}
