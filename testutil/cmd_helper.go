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

package testutil

import (
	"fmt"
	"os/exec"
	"strings"
)

type FakeCmd struct {
	expectedCommand string
	stdout          []byte
	err             error
}

func NewFakeCmd(expectedCommand string, err error) *FakeCmd {
	return &FakeCmd{
		expectedCommand: expectedCommand,
		err:             err,
	}
}

func NewFakeCmdOut(expectedCommand, stdout string, err error) *FakeCmd {
	return &FakeCmd{
		expectedCommand: expectedCommand,
		stdout:          []byte(stdout),
		err:             err,
	}
}

func (f *FakeCmd) RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	actualCommand := strings.Join(cmd.Args, " ")
	if f.expectedCommand != actualCommand {
		return nil, fmt.Errorf("Expected: %s. Got: %s", f.expectedCommand, actualCommand)
	}

	if f.stdout == nil {
		return nil, fmt.Errorf("Expected RunCmd(%s) to be called. Got RunCmdOut(%s)", f.expectedCommand, actualCommand)
	}

	return f.stdout, f.err
}

func (f *FakeCmd) RunCmd(cmd *exec.Cmd) error {
	actualCommand := strings.Join(cmd.Args, " ")
	if f.expectedCommand != actualCommand {
		return fmt.Errorf("Expected: %s. Got: %s", f.expectedCommand, actualCommand)
	}

	if f.stdout != nil {
		return fmt.Errorf("Expected RunCmdOut(%s) to be called. Got RunCmd(%s)", f.expectedCommand, actualCommand)
	}

	return f.err
}
