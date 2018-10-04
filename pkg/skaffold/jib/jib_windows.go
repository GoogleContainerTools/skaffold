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

package jib

import (
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func getCommand(workspace, defaultExecutable, wrapperExecutable string, defaultArgs []string) *exec.Cmd {
	executable := defaultExecutable
	args := defaultArgs

	if wrapperExecutable, err := util.AbsFile(workspace, wrapperExecutable); err == nil {
		executable = "cmd"
		args = append([]string{wrapperExecutable}, args...)
		args = append([]string{"/c"}, args...)
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = workspace
	return cmd
}
