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
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/alessio/shellescape"

	"sigs.k8s.io/kind/pkg/errors"
)

// PrettyCommand takes arguments identical to Cmder.Command,
// it returns a pretty printed command that could be pasted into a shell
func PrettyCommand(name string, args ...string) string {
	var out strings.Builder
	out.WriteString(shellescape.Quote(name))
	for _, arg := range args {
		out.WriteByte(' ')
		out.WriteString(shellescape.Quote(arg))
	}
	return out.String()
}

// RunErrorForError returns a RunError if the error contains a RunError.
// Otherwise it returns nil
func RunErrorForError(err error) *RunError {
	var runError *RunError
	for {
		if rErr, ok := err.(*RunError); ok {
			runError = rErr
		}
		if causerErr, ok := err.(errors.Causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}
	return runError
}

// CombinedOutputLines is like os/exec's cmd.CombinedOutput(),
// but over our Cmd interface, and instead of returning the byte buffer of
// stderr + stdout, it scans these for lines and returns a slice of output lines
func CombinedOutputLines(cmd Cmd) (lines []string, err error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	cmd.SetStderr(&buff)
	err = cmd.Run()
	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, err
}

// OutputLines is like os/exec's cmd.Output(),
// but over our Cmd interface, and instead of returning the byte buffer of
// stdout, it scans these for lines and returns a slice of output lines
func OutputLines(cmd Cmd) (lines []string, err error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	err = cmd.Run()
	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, err
}

// Output is like os/exec's cmd.Output, but over our Cmd interface
func Output(cmd Cmd) ([]byte, error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	err := cmd.Run()
	return buff.Bytes(), err
}

// InheritOutput sets cmd's output to write to the current process's stdout and stderr
func InheritOutput(cmd Cmd) Cmd {
	cmd.SetStderr(os.Stderr)
	cmd.SetStdout(os.Stdout)
	return cmd
}

// RunWithStdoutReader runs cmd with stdout piped to readerFunc
func RunWithStdoutReader(cmd Cmd, readerFunc func(io.Reader) error) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.SetStdout(pw)

	return errors.AggregateConcurrent([]func() error{
		func() error {
			defer pr.Close()
			return readerFunc(pr)
		},
		func() error {
			defer pw.Close()
			return cmd.Run()
		},
	})
}

// RunWithStdinWriter runs cmd with writerFunc piped to stdin
func RunWithStdinWriter(cmd Cmd, writerFunc func(io.Writer) error) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.SetStdin(pr)

	return errors.AggregateConcurrent([]func() error{
		func() error {
			defer pw.Close()
			return writerFunc(pw)
		},
		func() error {
			defer pr.Close()
			return cmd.Run()
		},
	})
}
