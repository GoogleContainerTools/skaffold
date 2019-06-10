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

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/spf13/pflag"
)

func TestNewCmdDescription(t *testing.T) {
	cmd := NewCmd(nil, "help").WithDescription("prints help").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
}

func TestNewCmdLongDescription(t *testing.T) {
	cmd := NewCmd(nil, "help").WithLongDescription("long description").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "long description", cmd.Long)
}

func TestNewCmdNoArgs(t *testing.T) {
	cmd := NewCmd(nil, "").NoArgs(nil)

	testutil.CheckError(t, false, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"extract arg"}))
}

func TestNewCmdExactArgs(t *testing.T) {
	cmd := NewCmd(nil, "").ExactArgs(1, nil)

	testutil.CheckError(t, true, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, false, cmd.Args(cmd, []string{"valid"}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"valid", "extra"}))
}

func TestNewCmdError(t *testing.T) {
	cmd := NewCmd(nil, "").NoArgs(func(out io.Writer) error {
		return errors.New("expected error")
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, true, err, "expected error", err.Error())
}

func TestNewCmdOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := NewCmd(&buf, "").ExactArgs(1, func(out io.Writer, args []string) error {
		fmt.Fprintf(out, "test output: %v\n", args)
		return nil
	})

	err := cmd.RunE(nil, []string{"arg1"})

	testutil.CheckErrorAndDeepEqual(t, false, err, "test output: [arg1]\n", buf.String())
}

func TestNewCmdWithFlags(t *testing.T) {
	cmd := NewCmd(nil, "").WithFlags(func(flagSet *pflag.FlagSet) {
		flagSet.Bool("test", false, "usage")
	}).NoArgs(nil)

	flags := listFlags(cmd.Flags())

	testutil.CheckDeepEqual(t, 1, len(flags))
	testutil.CheckDeepEqual(t, "usage", flags["test"].Usage)
}

func TestNewCmdWithCommonFlags(t *testing.T) {
	cmd := NewCmd(nil, "run").WithCommonFlags().NoArgs(nil)

	flags := listFlags(cmd.Flags())

	if _, present := flags["profile"]; !present {
		t.Error("Expected flag `profile` to be added")
	}
}

func listFlags(set *pflag.FlagSet) map[string]*pflag.Flag {
	flagsByName := make(map[string]*pflag.Flag)

	set.VisitAll(func(f *pflag.Flag) {
		flagsByName[f.Name] = f
	})

	return flagsByName
}
