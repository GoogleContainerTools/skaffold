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
	"testing"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHasCmdAnnotation(t *testing.T) {
	tests := []struct {
		description string
		cmd         string
		definedOn   []string
		expected    bool
	}{
		{
			description: "flag has command annotations",
			cmd:         "build",
			definedOn:   []string{"build", "events"},
			expected:    true,
		},
		{
			description: "flag does not have command annotations",
			cmd:         "build",
			definedOn:   []string{"some"},
		},
		{
			description: "flag has all annotations",
			cmd:         "build",
			definedOn:   []string{"all"},
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hasAnnotation := hasCmdAnnotation(test.cmd, test.definedOn)

			t.CheckDeepEqual(test.expected, hasAnnotation)
		})
	}
}

func TestAddFlagsSmoke(t *testing.T) {
	// Collect all commands that have common flags.
	commands := map[string]bool{}
	for _, fr := range flagRegistry {
		for _, command := range fr.DefinedOn {
			commands[command] = true
		}
	}

	// Make sure AddFlags() works for every command.
	for command := range commands {
		AddFlags(&cobra.Command{
			Use:   command,
			Short: "Test command for smoke testing",
		})
	}
}

func TestMakeFlag(t *testing.T) {
	var v string
	f := Flag{
		Name:          "flag",
		Shorthand:     "f",
		Value:         &v,
		Hidden:        true,
		FlagAddMethod: "StringVar",
		DefValue:      "default",
		DefValuePerCommand: map[string]interface{}{
			"debug": "dbg",
			"build": "bld",
		},
		NoOptDefVal: "nooptdefval",
	}

	testutil.Run(t, "just default value", func(t *testutil.T) {
		test := f.flag("test")
		t.CheckDeepEqual("flag", test.Name)
		t.CheckDeepEqual("f", test.Shorthand)
		t.CheckDeepEqual(true, test.Hidden)
		t.CheckDeepEqual("default", test.DefValue)
		t.CheckDeepEqual("nooptdefval", test.NoOptDefVal)
	})

	testutil.Run(t, "default value for debug", func(t *testutil.T) {
		debug := f.flag("debug")
		t.CheckDeepEqual("flag", debug.Name)
		t.CheckDeepEqual("f", debug.Shorthand)
		t.CheckDeepEqual(true, debug.Hidden)
		t.CheckDeepEqual("dbg", debug.DefValue)
		t.CheckDeepEqual("nooptdefval", debug.NoOptDefVal)
	})

	testutil.Run(t, "default value for build", func(t *testutil.T) {
		build := f.flag("build")
		t.CheckDeepEqual("flag", build.Name)
		t.CheckDeepEqual("f", build.Shorthand)
		t.CheckDeepEqual(true, build.Hidden)
		t.CheckDeepEqual("bld", build.DefValue)
		t.CheckDeepEqual("nooptdefval", build.NoOptDefVal)
	})
}

func TestResetFlagDefaults(t *testing.T) {
	var v string
	var sl []string

	valueFlag := Flag{
		Name:          "value",
		Value:         &v,
		FlagAddMethod: "StringVar",
		DefValue:      "default",
		DefValuePerCommand: map[string]interface{}{
			"debug": "dbg",
			"build": "bld",
		},
		DefinedOn: []string{"build", "debug", "test"},
	}
	sliceFlag := Flag{
		Name:          "slice",
		Value:         &sl,
		FlagAddMethod: "StringSliceVar",
		DefValue:      []string{"default"},
		DefValuePerCommand: map[string]interface{}{
			"debug": []string{"dbg", "other"},
			"build": []string{"bld"},
		},
		DefinedOn: []string{"build", "debug", "test"},
	}
	flagRegistry := []*Flag{&valueFlag, &sliceFlag}

	tests := []struct {
		command       string
		expectedValue string
		expectedSlice []string
	}{
		{"test", "default", []string{"default"}},
		{"debug", "dbg", []string{"dbg", "other"}},
		{"build", "bld", []string{"bld"}},
	}
	for _, test := range tests {
		testutil.Run(t, test.command, func(t *testutil.T) {
			cmd := cobra.Command{Use: test.command}
			for _, f := range flagRegistry {
				cmd.Flags().AddFlag(f.flag(test.command))
			}

			// ResetFlagDefaults should reset to defaults for the given command
			v = "randovalue"
			sl = []string{"rando", "value"}
			ResetFlagDefaults(&cmd, flagRegistry)

			t.CheckDeepEqual(v, test.expectedValue)
			t.CheckDeepEqual(sl, test.expectedSlice)
		})
	}
}
