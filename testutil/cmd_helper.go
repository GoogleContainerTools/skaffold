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

package testutil

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type FakeCmd struct {
	t           *testing.T
	runs        []run
	timesCalled int
}

type run struct {
	command    string
	input      []byte
	output     []byte
	env        []string
	dir        *string
	err        error
	pipeOutput bool
}

func newFakeCmd() *FakeCmd {
	return &FakeCmd{}
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

func (c *FakeCmd) ForTest(t *testing.T) {
	if c != nil {
		c.t = t
	}
}

func CmdRun(command string) *FakeCmd {
	return newFakeCmd().AndRun(command)
}

func CmdRunInputOut(command string, input string, output string) *FakeCmd {
	return newFakeCmd().AndRunInputOut(command, input, output)
}

func CmdRunErr(command string, err error) *FakeCmd {
	return newFakeCmd().AndRunErr(command, err)
}

func CmdRunOut(command string, output string) *FakeCmd {
	return newFakeCmd().AndRunOut(command, output)
}

func CmdRunDirOut(command string, dir string, output string) *FakeCmd {
	return newFakeCmd().AndRunDirOut(command, dir, output)
}

func CmdRunOutErr(command string, output string, err error) *FakeCmd {
	return newFakeCmd().AndRunOutErr(command, output, err)
}

func CmdRunEnv(command string, env []string) *FakeCmd {
	return newFakeCmd().AndRunEnv(command, env)
}

// CmdRunWithOutput programs the fake runner with a command and expected output
func CmdRunWithOutput(command, output string) *FakeCmd {
	return newFakeCmd().AndRunWithOutput(command, output)
}

func (c *FakeCmd) AndRun(command string) *FakeCmd {
	return c.addRun(run{
		command: command,
	})
}

func (c *FakeCmd) AndRunInput(command, input string) *FakeCmd {
	return c.addRun(run{
		command: command,
		input:   []byte(input),
	})
}

func (c *FakeCmd) AndRunErr(command string, err error) *FakeCmd {
	return c.addRun(run{
		command: command,
		err:     err,
	})
}

// AndRunWithOutput takes a command and an expected output.
// It expected to match up with a call to RunCmd, and pipes
// the provided output to RunCmd's exec.Cmd's stdout.
func (c *FakeCmd) AndRunWithOutput(command, output string) *FakeCmd {
	return c.addRun(run{
		command:    command,
		output:     []byte(output),
		pipeOutput: true,
	})
}

func (c *FakeCmd) AndRunInputOut(command string, input string, output string) *FakeCmd {
	return c.addRun(run{
		command: command,
		input:   []byte(input),
		output:  []byte(output),
	})
}

func (c *FakeCmd) AndRunOut(command string, output string) *FakeCmd {
	return c.addRun(run{
		command: command,
		output:  []byte(output),
	})
}

func (c *FakeCmd) AndRunDirOut(command string, dir string, output string) *FakeCmd {
	return c.addRun(run{
		command: command,
		dir:     &dir,
		output:  []byte(output),
	})
}

func (c *FakeCmd) AndRunOutErr(command string, output string, err error) *FakeCmd {
	return c.addRun(run{
		command: command,
		output:  []byte(output),
		err:     err,
	})
}

func (c *FakeCmd) AndRunEnv(command string, env []string) *FakeCmd {
	return c.addRun(run{
		command: command,
		env:     env,
	})
}

func (c *FakeCmd) RunCmdOut(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	c.timesCalled++
	command := strings.Join(cmd.Args, " ")

	r, err := c.popRun()
	if err != nil {
		c.t.Fatalf("unable to run RunCmdOut() with command %q: %v", command, err)
	}

	if r.command != command {
		c.t.Errorf("expected: %s. Got: %s", r.command, command)
	}

	c.assertCmdEnv(r.env, cmd.Env)
	c.assertCmdDir(r.dir, cmd.Dir)

	if err := c.assertInput(cmd, r, command); err != nil {
		return nil, err
	}

	if r.output == nil {
		c.t.Errorf("expected RunCmd(%s) to be called. Got RunCmdOut(%s)", r.command, command)
	}

	return r.output, r.err
}

func (c *FakeCmd) RunCmd(ctx context.Context, cmd *exec.Cmd) error {
	c.timesCalled++
	command := strings.Join(cmd.Args, " ")

	r, err := c.popRun()
	if err != nil {
		c.t.Fatalf("unable to run RunCmd() with command %q", command)
	}

	if r.command != command {
		c.t.Errorf("\nwanted: %s\n\n   got: %s", r.command, command)
	}

	if r.output != nil {
		if !r.pipeOutput {
			c.t.Errorf("expected RunCmdOut(%s) to be called. Got RunCmd(%s)", r.command, command)
		} else {
			cmd.Stdout.Write(r.output)
		}
	}

	c.assertCmdEnv(r.env, cmd.Env)

	if err := c.assertInput(cmd, r, command); err != nil {
		return err
	}

	return r.err
}

func (c *FakeCmd) assertInput(cmd *exec.Cmd, r *run, command string) error {
	if r.input == nil {
		return nil
	}

	if cmd.Stdin == nil {
		c.t.Error("expected to run the command with a custom stdin", command)
		return nil
	}

	buf, err := io.ReadAll(cmd.Stdin)
	if err != nil {
		return err
	}

	actualInput := string(buf)
	expectedInput := string(r.input)
	if actualInput != expectedInput {
		c.t.Errorf("wrong input. Expected: %s. Got %s", expectedInput, actualInput)
	}

	return nil
}

// assertCmdEnv ensures that actualEnv contains all values from requiredEnv
func (c *FakeCmd) assertCmdEnv(requiredEnv, actualEnv []string) {
	if requiredEnv == nil {
		return
	}
	c.t.Helper()

	envs := make(map[string]string, len(actualEnv))
	for _, e := range actualEnv {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			c.t.Fatal("invalid environment: missing '=' in:", e)
		}
		envs[kv[0]] = kv[1]
	}

	for _, e := range requiredEnv {
		kv := strings.SplitN(e, "=", 2)
		value, found := envs[kv[0]]
		switch len(kv) {
		case 1:
			if found {
				c.t.Errorf("wanted env variable %q with value %q: env=%v", kv[0], kv[1], actualEnv)
			}

		case 2:
			if !found {
				c.t.Errorf("wanted env variable %q with value %q: env=%v", kv[0], kv[1], actualEnv)
			} else if value != kv[1] {
				c.t.Errorf("expected env variable %q to be %q but found %q", kv[0], kv[1], value)
			}
		}
	}
}

// assertCmdDir ensures that actualDir contains matches requiredDir
func (c *FakeCmd) assertCmdDir(requiredDir *string, actualDir string) {
	if requiredDir == nil {
		return
	}
	c.t.Helper()

	if *requiredDir != actualDir {
		c.t.Errorf("expected: %s. Got: %s", *requiredDir, actualDir)
	}
}

func (c *FakeCmd) TimesCalled() int {
	return c.timesCalled
}
