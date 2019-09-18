// +build !windows

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

package util

import (
	"context"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// CreateCommand creates an `exec.Cmd` that is configured to call the
// executable (possibly using a wrapper in `workingDir`, when found) with the given arguments,
// with working directory set to `workingDir`.
func (cw CommandWrapper) CreateCommand(ctx context.Context, workingDir string, args []string) exec.Cmd {
	executable := cw.Executable

	if cw.Wrapper != "" && !SkipWrapperCheck {
		if wrapperExecutable, err := AbsFile(workingDir, cw.Wrapper); err == nil {
			logrus.Debugf("Using wrapper for %s: %s", cw.Wrapper, cw.Executable)
			executable = wrapperExecutable
		}
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = workingDir
	return *cmd
}
