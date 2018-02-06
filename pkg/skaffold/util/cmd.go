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

package util

import (
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/pkg/errors"
)

// DefaultExecCommand runs commands using exec.Cmd
var DefaultExecCommand Command

func init() {
	DefaultExecCommand = &Commander{}
}

func ResetDefaultExecCommand() {
	DefaultExecCommand = &Commander{}
}

// Command is an interface used to run commands. All packages should use this
// interface instead of calling exec.Cmd directly.
type Command interface {
	RunCommand(cmd *exec.Cmd, stdin io.Reader) ([]byte, []byte, error)
}

func RunCommand(cmd *exec.Cmd, stdin io.Reader) ([]byte, []byte, error) {
	return DefaultExecCommand.RunCommand(cmd, stdin)
}

// Commander is the exec.Cmd implementation of the Command interface
type Commander struct{}

// RunCommand runs an exec.Command, optionally reading from stdin and return
// the stdout, stderr, and error responses respectively.
func (*Commander) RunCommand(cmd *exec.Cmd, stdin io.Reader) ([]byte, []byte, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if stdin != nil {
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		go func() {
			defer stdinPipe.Close()
			io.Copy(stdinPipe, stdin)
		}()
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Wrapf(err, "starting command %v", cmd)
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return nil, nil, err
	}
	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Wait(); err != nil {
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}
