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

package helper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNoArgCommandDescription(t *testing.T) {
	cmd := NoArgCommand(nil, "help", "prints help", nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
}

func TestNoArgCommandError(t *testing.T) {
	cmd := NoArgCommand(ioutil.Discard, "", "", func(out io.Writer) error {
		return errors.New("expected error")
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, true, err, "expected error", err.Error())
}

func TestNoArgCommandOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := NoArgCommand(&buf, "", "", func(out io.Writer) error {
		fmt.Fprintln(out, "test output")
		return nil
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, "test output\n", buf.String())
}

func TestNoArgCommandValidation(t *testing.T) {
	cmd := NoArgCommand(nil, "", "", nil)

	testutil.CheckError(t, false, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"extract arg"}))
}

func TestArgsCommandDescription(t *testing.T) {
	cmd := ArgsCommand(nil, "help", "prints help", 0, nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
}

func TestArgsCommandError(t *testing.T) {
	cmd := ArgsCommand(ioutil.Discard, "", "", 0, func(out io.Writer, _ []string) error {
		return errors.New("expected error")
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, true, err, "expected error", err.Error())
}

func TestArgsCommandOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := ArgsCommand(&buf, "", "", 0, func(out io.Writer, args []string) error {
		fmt.Fprintf(out, "test output: %v\n", args)
		return nil
	})

	err := cmd.RunE(nil, []string{"arg1"})

	testutil.CheckErrorAndDeepEqual(t, false, err, "test output: [arg1]\n", buf.String())
}

func TestArgsCommandValidation(t *testing.T) {
	cmd := ArgsCommand(nil, "", "", 1, nil)

	testutil.CheckError(t, true, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, false, cmd.Args(cmd, []string{"valid"}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"valid", "extra"}))
}
