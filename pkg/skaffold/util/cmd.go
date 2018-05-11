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
	"io/ioutil"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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
		return nil, errors.Wrapf(err, "starting command %v", cmd)
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return nil, err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return stdout, errors.Wrapf(err, "Running %s: stdout %s, stderr: %s, err: %v", cmd.Args, stdout, stderr, err)
	}

	logrus.Debugf("Command output: stdout %s, stderr: %s", stdout, stderr)

	return stdout, nil
}

// RunCmd runs an exec.Command.
func (*Commander) RunCmd(cmd *exec.Cmd) error {
	logrus.Debugf("Running command: %s", cmd.Args)
	return cmd.Run()
}
