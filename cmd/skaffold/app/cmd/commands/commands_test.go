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

package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewCommandDescription(t *testing.T) {
	cmd := New(nil).WithDescription("help", "prints help").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
}

func TestNewCommandLongDescription(t *testing.T) {
	cmd := New(nil).WithLongDescription("help", "prints help", "long description").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
	testutil.CheckDeepEqual(t, "long description", cmd.Long)
}

func TestNewCommandNoArgs(t *testing.T) {
	cmd := New(nil).NoArgs(nil)

	testutil.CheckError(t, false, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"extract arg"}))
}

func TestNewCommandExactArgs(t *testing.T) {
	cmd := New(nil).ExactArgs(1, nil)

	testutil.CheckError(t, true, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, false, cmd.Args(cmd, []string{"valid"}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"valid", "extra"}))
}

func TestNewCommandError(t *testing.T) {
	cmd := New(nil).NoArgs(func(out io.Writer) error {
		return errors.New("expected error")
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, true, err, "expected error", err.Error())
}

func TestNewCommandOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := New(&buf).ExactArgs(1, func(out io.Writer, args []string) error {
		fmt.Fprintf(out, "test output: %v\n", args)
		return nil
	})

	err := cmd.RunE(nil, []string{"arg1"})

	testutil.CheckErrorAndDeepEqual(t, false, err, "test output: [arg1]\n", buf.String())
}
