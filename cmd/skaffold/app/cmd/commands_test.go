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
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewCmdDescription(t *testing.T) {
	cmd := NewCmd("help").WithDescription("prints help").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "prints help", cmd.Short)
}

func TestNewCmdLongDescription(t *testing.T) {
	cmd := NewCmd("help").WithLongDescription("long description").NoArgs(nil)

	testutil.CheckDeepEqual(t, "help", cmd.Use)
	testutil.CheckDeepEqual(t, "long description", cmd.Long)
}

func TestNewCmdExample(t *testing.T) {
	cmd := NewCmd("").WithExample("comment1", "dev --flag1").NoArgs(nil)

	testutil.CheckDeepEqual(t, "  # comment1\n  skaffold dev --flag1\n", cmd.Example)
}

func TestNewCmdExamples(t *testing.T) {
	cmd := NewCmd("").WithExample("comment1", "run --flag1").WithExample("comment2", "run --flag2").NoArgs(nil)

	testutil.CheckDeepEqual(t, "  # comment1\n  skaffold run --flag1\n\n  # comment2\n  skaffold run --flag2\n", cmd.Example)
}

func TestNewCmdNoArgs(t *testing.T) {
	cmd := NewCmd("").NoArgs(nil)

	testutil.CheckError(t, false, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"extract arg"}))
}

func TestNewCmdExactArgs(t *testing.T) {
	cmd := NewCmd("").ExactArgs(1, nil)

	testutil.CheckError(t, true, cmd.Args(cmd, []string{}))
	testutil.CheckError(t, false, cmd.Args(cmd, []string{"valid"}))
	testutil.CheckError(t, true, cmd.Args(cmd, []string{"valid", "extra"}))
}

func TestNewCmdError(t *testing.T) {
	cmd := NewCmd("").NoArgs(func(ctx context.Context, out io.Writer) error {
		return errors.New("expected error")
	})

	err := cmd.RunE(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, true, err, "expected error", err.Error())
}

func TestNewCmdOutput(t *testing.T) {
	var buf bytes.Buffer

	cmd := NewCmd("").ExactArgs(1, func(ctx context.Context, out io.Writer, args []string) error {
		fmt.Fprintf(out, "test output: %v\n", args)
		return nil
	})
	cmd.SetOutput(&buf)

	err := cmd.RunE(nil, []string{"arg1"})

	testutil.CheckErrorAndDeepEqual(t, false, err, "test output: [arg1]\n", buf.String())
}

func TestNewCmdWithFlags(t *testing.T) {
	cmd := NewCmd("").WithFlags(func(flagSet *pflag.FlagSet) {
		flagSet.Bool("test", false, "usage")
	}).NoArgs(nil)

	flags := listFlags(cmd.Flags())

	testutil.CheckDeepEqual(t, 1, len(flags))
	testutil.CheckDeepEqual(t, "usage", flags["test"].Usage)
}

func TestNewCmdWithCommonFlags(t *testing.T) {
	cmd := NewCmd("run").WithCommonFlags().NoArgs(nil)

	flags := listFlags(cmd.Flags())

	if _, present := flags["profile"]; !present {
		t.Error("Expected flag `profile` to be added")
	}
}

func TestNewCmdHidden(t *testing.T) {
	cmd := NewCmd("").NoArgs(nil)
	testutil.CheckDeepEqual(t, false, cmd.Hidden)

	cmd = NewCmd("").Hidden().NoArgs(nil)
	testutil.CheckDeepEqual(t, true, cmd.Hidden)
}

func listFlags(set *pflag.FlagSet) map[string]*pflag.Flag {
	flagsByName := make(map[string]*pflag.Flag)

	set.VisitAll(func(f *pflag.Flag) {
		flagsByName[f.Name] = f
	})

	return flagsByName
}
