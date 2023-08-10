/*
Copyright 2018 The Kubernetes Authors.

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
	"bytes"
	"context"
	"io"
	osexec "os/exec"
	"sync"

	"sigs.k8s.io/kind/pkg/errors"
)

// LocalCmd wraps os/exec.Cmd, implementing the kind/pkg/exec.Cmd interface
type LocalCmd struct {
	*osexec.Cmd
}

var _ Cmd = &LocalCmd{}

// LocalCmder is a factory for LocalCmd, implementing Cmder
type LocalCmder struct{}

var _ Cmder = &LocalCmder{}

// Command returns a new exec.Cmd backed by Cmd
func (c *LocalCmder) Command(name string, arg ...string) Cmd {
	return &LocalCmd{
		Cmd: osexec.Command(name, arg...),
	}
}

// CommandContext is like Command but includes a context
func (c *LocalCmder) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	return &LocalCmd{
		Cmd: osexec.CommandContext(ctx, name, arg...),
	}
}

// SetEnv sets env
func (cmd *LocalCmd) SetEnv(env ...string) Cmd {
	cmd.Env = env
	return cmd
}

// SetStdin sets stdin
func (cmd *LocalCmd) SetStdin(r io.Reader) Cmd {
	cmd.Stdin = r
	return cmd
}

// SetStdout set stdout
func (cmd *LocalCmd) SetStdout(w io.Writer) Cmd {
	cmd.Stdout = w
	return cmd
}

// SetStderr sets stderr
func (cmd *LocalCmd) SetStderr(w io.Writer) Cmd {
	cmd.Stderr = w
	return cmd
}

// Run runs the command
// If the returned error is non-nil, it should be of type *RunError
func (cmd *LocalCmd) Run() error {
	// Background:
	// Go's stdlib will setup and use a shared fd when cmd.Stderr == cmd.Stdout
	// In any other case, it will use different fds, which will involve
	// two different io.Copy goroutines writing to cmd.Stderr and cmd.Stdout
	//
	// Given this, we must synchronize capturing the output to a buffer
	// IFF ! interfaceEqual(cmd.Sterr, cmd.Stdout)
	var combinedOutput bytes.Buffer
	var combinedOutputWriter io.Writer = &combinedOutput
	if cmd.Stdout == nil && cmd.Stderr == nil {
		// Case 1: If stdout and stderr are nil, we can just use the buffer
		// The buffer will be == and Go will use one fd / goroutine
		cmd.Stdout = combinedOutputWriter
		cmd.Stderr = combinedOutputWriter
	} else if interfaceEqual(cmd.Stdout, cmd.Stderr) {
		// Case 2: If cmd.Stdout == cmd.Stderr go will still share the fd,
		// but we need to wrap with a MultiWriter to respect the other writer
		// and our buffer.
		// The MultiWriter will be == and Go will use one fd / goroutine
		cmd.Stdout = io.MultiWriter(cmd.Stdout, combinedOutputWriter)
		cmd.Stderr = cmd.Stdout
	} else {
		// Case 3: If cmd.Stdout != cmd.Stderr, we need to synchronize the
		// combined output writer.
		// Go will use different fds / write routines for stdout and stderr
		combinedOutputWriter = &mutexWriter{
			writer: &combinedOutput,
		}
		// wrap writers if non-nil
		if cmd.Stdout != nil {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, combinedOutputWriter)
		} else {
			cmd.Stdout = combinedOutputWriter
		}
		if cmd.Stderr != nil {
			cmd.Stderr = io.MultiWriter(cmd.Stderr, combinedOutputWriter)
		} else {
			cmd.Stderr = combinedOutputWriter
		}
	}
	// TODO: should be in the caller or logger should be injected somehow ...
	if err := cmd.Cmd.Run(); err != nil {
		return errors.WithStack(&RunError{
			Command: cmd.Args,
			Output:  combinedOutput.Bytes(),
			Inner:   err,
		})
	}
	return nil
}

// interfaceEqual protects against panics from doing equality tests on
// two interfaces with non-comparable underlying types.
// This trivial is borrowed from the go stdlib in os/exec
// Note that the recover will only happen if a is not comparable to b,
// in which case we'll return false
// We've lightly modified this to pass errcheck (explicitly ignoring recover)
func interfaceEqual(a, b interface{}) bool {
	defer func() {
		_ = recover()
	}()
	return a == b
}

// mutexWriter is a simple synchronized wrapper around an io.Writer
type mutexWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

func (m *mutexWriter) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err := m.writer.Write(b)
	return n, err
}
