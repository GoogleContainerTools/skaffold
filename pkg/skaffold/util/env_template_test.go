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

package util

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestEnvTemplate_ExecuteEnvTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]string
		env         []string
		want        string
		shouldErr   bool
	}{
		{
			description: "custom only",
			template:    "{{.FOO}}:{{.BAR}}",
			customMap: map[string]string{
				"FOO": "foo",
				"BAR": "bar",
			},
			want: "foo:bar",
		},
		{
			description: "env only",
			template:    "{{.FOO}}-{{.BAZ}}:latest",
			env:         []string{"FOO=BAR", "BAZ=BAT"},
			want:        "BAR-BAT:latest",
		},
		{
			description: "both and custom precedence",
			template:    "{{.MY_NAME}}-{{.FROM_ENV}}:latest",
			env:         []string{"FROM_ENV=FOO", "MY_NAME=BAR"},
			customMap: map[string]string{
				"FOO":     "foo",
				"MY_NAME": "from_custom",
			},
			want: "from_custom-FOO:latest",
		},
		{
			description: "both and custom precedence",
			template:    "{{with $x := nil}}tag{{end}}",
			env:         []string{"VAL=KEY"},
			shouldErr:   true,
		},
		{
			description: "missing results in empty",
			template:    `{{default "a" .FOO}}:{{.BAR}}`,
			customMap:   map[string]string{},
			want:        "a:<no value>",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&OSEnviron, func() []string { return test.env })

			testTemplate, err := ParseEnvTemplate(test.template)
			t.CheckNoError(err)

			got, err := ExecuteEnvTemplate(testTemplate, test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.want, got)

			got, err = ExpandEnvTemplate(test.template, test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.want, got)
		})
	}
}

func TestEnvTemplate_ExpandEnvTemplateOrFail(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]string
		env         []string
		want        string
		shouldErr   bool
	}{
		{
			description: "env and custom precedence",
			template:    "{{.MY_NAME}}-{{.FROM_ENV}}:latest",
			env:         []string{"FROM_ENV=FOO", "MY_NAME=BAR"},
			customMap: map[string]string{
				"FOO":     "foo",
				"MY_NAME": "from_custom",
			},
			want: "from_custom-FOO:latest",
		},
		{
			description: "variable does not exist",
			template:    "{{.DOES_NOT_EXIST}}",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&OSEnviron, func() []string { return test.env })
			got, err := ExpandEnvTemplateOrFail(test.template, test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.want, got)
		})
	}
}

func TestMapToFlag(t *testing.T) {
	foo := "foo"
	bar := "bar"
	type args struct {
		m    map[string]*string
		flag string
	}
	tests := []struct {
		description string
		args        args
		want        []string
		wantErr     bool
	}{
		{
			description: "All keys have value",
			args: args{
				m: map[string]*string{
					"FOO": &foo,
					"BAR": &bar,
				},
				flag: "--flag",
			},
			want:    []string{"--flag", "BAR=bar", "--flag", "FOO=foo"},
			wantErr: false,
		},
		{
			description: "Only keys",
			args: args{
				m: map[string]*string{
					"FOO": nil,
					"BAR": nil,
				},
				flag: "--flag",
			},
			want:    []string{"--flag", "BAR", "--flag", "FOO"},
			wantErr: false,
		},
		{
			description: "Mixed",
			args: args{
				m: map[string]*string{
					"FOO": &foo,
					"BAR": nil,
				},
				flag: "--flag",
			},
			want:    []string{"--flag", "BAR", "--flag", "FOO=foo"},
			wantErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := MapToFlag(test.args.m, test.args.flag)
			t.CheckNoError(err)
			t.CheckErrorAndDeepEqual(test.wantErr, err, test.want, got)
		})
	}
}

func TestDefaultFunc(t *testing.T) {
	for _, empty := range []interface{}{nil, false, 0, "", []string{}} {
		t.Run(fmt.Sprintf("empties: %v (%T)", empty, empty), func(t *testing.T) {
			dflt := "default"
			if defaultFunc(dflt, empty) != dflt {
				t.Error("did not return default")
			}
		})
	}
	s := "string"
	for _, nonEmpty := range []interface{}{&s, true, 1, "hoot", []string{"hoot"}} {
		t.Run(fmt.Sprintf("non-empty: %v (%T)", nonEmpty, nonEmpty), func(t *testing.T) {
			dflt := "default"
			if defaultFunc(dflt, nonEmpty) == dflt {
				t.Error("should not return default")
			}
		})
	}
}

func TestRunCmdFunc(t *testing.T) {
	tests := []struct {
		description     string
		commandName     string
		args            []string
		output          string
		expectedCommand string
		err             error
	}{
		{
			description:     "test running command succeeds",
			commandName:     "bash",
			args:            []string{"-c", "git rev-parse --verify HEAD"},
			output:          "123",
			expectedCommand: "bash -c git rev-parse --verify HEAD",
		},
		{
			description:     "test running command fails",
			commandName:     "bash",
			args:            []string{"-c", "gib rev-parse --verify HEAD"},
			output:          "",
			expectedCommand: "bash -c gib rev-parse --verify HEAD",
			err:             fmt.Errorf("command not found"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultExecCommand, testutil.CmdRunOut(test.expectedCommand, test.output))
			out, _ := runCmdFunc(test.commandName, test.args...)
			t.CheckErrorAndDeepEqual(test.err != nil, test.err, test.output, out)
		})
	}
}
