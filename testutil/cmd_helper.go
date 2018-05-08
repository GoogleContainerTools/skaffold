/*
Copyright 2018 Google LLC

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

package testutil

import (
	"fmt"
	"os/exec"
	"strings"
)

type FakeRunCommand struct {
	expectedCommand string
	stdout          string
	stderr          string
	err             error
}

func NewFakeRunCommand(expectedCommand, stdout, stderr string, err error) *FakeRunCommand {
	return &FakeRunCommand{
		expectedCommand: expectedCommand,
		stdout:          stdout,
		stderr:          stderr,
		err:             err,
	}
}

func (f *FakeRunCommand) RunCommand(cmd *exec.Cmd) ([]byte, []byte, error) {
	actualCommand := strings.Join(cmd.Args, " ")
	if f.expectedCommand != actualCommand {
		return nil, nil, fmt.Errorf("Expected: %s. Got: %s", f.expectedCommand, actualCommand)
	}

	return []byte(f.stdout), []byte(f.stderr), f.err
}
