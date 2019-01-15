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
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

type FakeCmd struct {
	t    *testing.T
	runs []run
}

type run struct {
	command string
	input   []byte
	output  []byte
	err     error
}

func NewFakeCmd(t *testing.T) *FakeCmd {
	return &FakeCmd{
		t: t,
	}
}

func (c *FakeCmd) addRun(r run) *FakeCmd {
	c.runs = append(c.runs, r)
	return c
}

func (c *FakeCmd) popRun() (*run, error) {
	if len(c.runs) == 0 {
		return nil, errors.New("no more run is expected")
	}

	run := c.runs[0]
	c.runs = c.runs[1:]
	return &run, nil
}

func (c *FakeCmd) WithRun(command string) *FakeCmd {
	return c.addRun(run{
		command: command,
	})
}

func (c *FakeCmd) WithRunInput(command, input string) *FakeCmd {
	return c.addRun(run{
		command: command,
		input:   []byte(input),
	})
}

func (c *FakeCmd) WithRunErr(command string, err error) *FakeCmd {
	return c.addRun(run{
		command: command,
		err:     err,
	})
}

func (c *FakeCmd) WithRunOut(command string, output string) *FakeCmd {
	return c.addRun(run{
		command: command,
		output:  []byte(output),
	})
}

func (c *FakeCmd) WithRunOutErr(command string, output string, err error) *FakeCmd {
	return c.addRun(run{
		command: command,
		output:  []byte(output),
		err:     err,
	})
}

func (c *FakeCmd) RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	command := strings.Join(cmd.Args, " ")

	r, err := c.popRun()
	if err != nil {
		c.t.Fatalf("unable to run RunCmdOut() with command %q", command)
	}

	if r.command != command {
		c.t.Errorf("expected: %s. Got: %s", r.command, command)
	}

	if r.output == nil {
		c.t.Errorf("expected RunCmd(%s) to be called. Got RunCmdOut(%s)", r.command, command)
	}

	return r.output, r.err
}

func (c *FakeCmd) RunCmd(cmd *exec.Cmd) error {
	command := strings.Join(cmd.Args, " ")

	r, err := c.popRun()
	if err != nil {
		c.t.Fatalf("unable to run RunCmd() with command %q", command)
	}

	if r.command != command {
		c.t.Errorf("expected: %s. Got: %s", r.command, command)
	}

	if r.output != nil {
		c.t.Errorf("expected RunCmdOut(%s) to be called. Got RunCmd(%s)", r.command, command)
	}

	if r.input != nil {
		if cmd.Stdin == nil {
			c.t.Error("expected to run the command with a custom stdin", command)
		}

		buf, err := ioutil.ReadAll(cmd.Stdin)
		if err != nil {
			return err
		}

		actualInput := string(buf)
		expectedInput := string(r.input)
		if actualInput != expectedInput {
			c.t.Errorf("wrong input. Expected: %s. Got %s", expectedInput, actualInput)
		}
	}

	return r.err
}
