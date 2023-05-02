/*
Copyright 2019 The Kubernetes Authors.

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

package exec

import (
	"context"
	"fmt"
	"io"
)

// Cmd abstracts over running a command somewhere, this is useful for testing
type Cmd interface {
	// Run executes the command (like os/exec.Cmd.Run), it should return
	// a *RunError if there is any error
	Run() error
	// Each entry should be of the form "key=value"
	SetEnv(...string) Cmd
	SetStdin(io.Reader) Cmd
	SetStdout(io.Writer) Cmd
	SetStderr(io.Writer) Cmd
}

// Cmder abstracts over creating commands
type Cmder interface {
	// command, args..., just like os/exec.Cmd
	Command(string, ...string) Cmd
	CommandContext(context.Context, string, ...string) Cmd
}

// RunError represents an error running a Cmd
type RunError struct {
	Command []string // [Name Args...]
	Output  []byte   // Captured Stdout / Stderr of the command
	Inner   error    // Underlying error if any
}

var _ error = &RunError{}

func (e *RunError) Error() string {
	// TODO(BenTheElder): implement formatter, and show output for %+v ?
	return fmt.Sprintf("command \"%s\" failed with error: %v", e.PrettyCommand(), e.Inner)
}

// PrettyCommand pretty prints the command in a way that could be pasted
// into a shell
func (e *RunError) PrettyCommand() string {
	return PrettyCommand(e.Command[0], e.Command[1:]...)
}

// Cause mimics github.com/pkg/errors's Cause pattern for errors
func (e *RunError) Cause() error {
	if e.Inner != nil {
		return e.Inner
	}
	return e
}
