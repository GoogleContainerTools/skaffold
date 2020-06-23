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
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/sirupsen/logrus"
)

type cmdError struct {
	args   []string
	stdout []byte
	stderr []byte
	cause  error
}

func (e *cmdError) Error() string {
	return fmt.Sprintf("running %s\n - stdout: %q\n - stderr: %q\n - cause: %s", e.args, e.stdout, e.stderr, e.cause)
}

func (e *cmdError) Unwrap() error {
	return e.cause
}

func (e *cmdError) ExitCode() int {
	if exitError, ok := e.cause.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return 0
}

// DefaultExecCommand runs commands using exec.Cmd
var DefaultExecCommand Command = &Commander{}

// Command is an interface used to run commands. All packages should use this
// interface instead of calling exec.Cmd directly.
type Command interface {
	RunCmdOut(cmd *exec.Cmd) ([]byte, error)
	RunCmd(cmd *exec.Cmd) error
}

func RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	return DefaultExecCommand.RunCmdOut(cmd)
}

func RunCmd(cmd *exec.Cmd) error {
	return DefaultExecCommand.RunCmd(cmd)
}

// Commander is the exec.Cmd implementation of the Command interface
type Commander struct{}

// RunCmdOut runs an exec.Command and returns the stdout and error.
func (*Commander) RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	logrus.Debugf("Running command: %s", cmd.Args)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting command %v: %w", cmd, err)
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return nil, err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return stdout, &cmdError{
			args:   cmd.Args,
			stdout: stdout,
			stderr: stderr,
			cause:  err,
		}
	}

	if len(stderr) > 0 {
		logrus.Debugf("Command output: [%s], stderr: %s", stdout, stderr)
	} else {
		logrus.Debugf("Command output: [%s]", stdout)
	}

	return stdout, nil
}

// RunCmd runs an exec.Command.
func (*Commander) RunCmd(cmd *exec.Cmd) error {
	logrus.Debugf("Running command: %s", cmd.Args)
	return cmd.Run()
}
