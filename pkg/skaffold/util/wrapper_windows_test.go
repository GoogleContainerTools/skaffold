// +build windows

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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetCommand(t *testing.T) {
	tests := []struct {
		description       string
		defaultExecutable string
		wrapperExecutable string
		args              []string
		filesInWorkspace  []string
		expectedCmd       func(workspace string) *exec.Cmd
	}{
		{
			description:       "wrapper not present",
			defaultExecutable: "executable",
			wrapperExecutable: "does-not-exist",
			args:              []string{"arg1", "arg2"},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				cmd := exec.CommandContext(context.TODO(), "executable", "arg1", "arg2")
				cmd.Dir = workspace
				return cmd
			},
		},
		{
			description:       "wrapper is present",
			defaultExecutable: "executable",
			wrapperExecutable: "wrapper",
			args:              []string{"arg1", "arg2"},
			filesInWorkspace:  []string{"wrapper.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				wrapper, err := AbsFile(workspace, "wrapper.bat")
				testutil.CheckError(t, false, err)
				cmd := exec.CommandContext(context.TODO(), "cmd", "/c", wrapper, "arg1", "arg2")
				cmd.Dir = workspace
				return cmd
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Touch(test.filesInWorkspace...)

			definition := &CommandWrapper{Executable: test.defaultExecutable, Wrapper: test.wrapperExecutable}
			cmd := definition.CreateCommand(context.TODO(), tmpDir.Root(), test.args)

			expectedCmd := test.expectedCmd(tmpDir.Root())
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}
