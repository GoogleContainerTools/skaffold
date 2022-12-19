/*
Copyright 2020 The Skaffold Authors

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

package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestStringOrUndefinedUsage(t *testing.T) {
	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.Flags().Var(&StringOrUndefined{}, "flag", "use it like this")
	cmd.SetOutput(&output)
	cmd.Usage()

	testutil.CheckDeepEqual(t, "Usage:\n\nFlags:\n      --flag string   use it like this\n", output.String())
}

func TestStringOrUndefined_SetNil(t *testing.T) {
	var s StringOrUndefined
	s.Set("hello")
	testutil.CheckDeepEqual(t, "hello", s.String())
	s.SetNil()
	testutil.CheckDeepEqual(t, "", s.String())
	testutil.CheckDeepEqual(t, (*string)(nil), s.value)
	testutil.CheckDeepEqual(t, (*string)(nil), s.Value())
}

func TestStringOrUndefined(t *testing.T) {
	tests := []struct {
		description string
		args        []string
		expected    *string
	}{
		{
			description: "undefined",
			args:        []string{},
			expected:    nil,
		},
		{
			description: "set",
			args:        []string{"--flag=value"},
			expected:    util.Ptr("value"),
		},
		{
			description: "empty",
			args:        []string{"--flag="},
			expected:    util.Ptr(""),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var flag StringOrUndefined

			cmd := &cobra.Command{}
			cmd.Flags().Var(&flag, "flag", "")
			cmd.SetArgs(test.args)
			cmd.Execute()

			t.CheckDeepEqual(test.expected, flag.value)
		})
	}
}

func TestBoolOrUndefinedUsage(t *testing.T) {
	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.Flags().Var(&BoolOrUndefined{}, "bool-flag", "use it like this")
	cmd.SetOut(&output)
	cmd.Usage()

	testutil.CheckDeepEqual(t, "Usage:\n\nFlags:\n      --bool-flag   use it like this\n", output.String())
}

func TestBoolOrUndefined_SetNil(t *testing.T) {
	var s BoolOrUndefined
	s.Set("false")
	testutil.CheckDeepEqual(t, "false", s.String())
	s.SetNil()
	testutil.CheckDeepEqual(t, "", s.String())
	testutil.CheckDeepEqual(t, (*bool)(nil), s.value)
	testutil.CheckDeepEqual(t, (*bool)(nil), s.Value())
}

func TestBoolOrUndefined(t *testing.T) {
	tests := []struct {
		description string
		args        []string
		expected    *bool
	}{
		{
			description: "undefined",
			args:        []string{},
			expected:    nil,
		},
		{
			description: "empty",
			args:        []string{"--bool-flag="},
			expected:    nil,
		},
		{
			description: "invalid",
			args:        []string{"--bool-flag=invalid"},
			expected:    nil,
		},
		{
			description: "true",
			args:        []string{"--bool-flag=true"},
			expected:    util.Ptr(true),
		},
		{
			description: "false",
			args:        []string{"--bool-flag=false"},
			expected:    util.Ptr(false),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var flag BoolOrUndefined

			cmd := &cobra.Command{}
			cmd.Flags().Var(&flag, "bool-flag", "")
			cmd.SetArgs(test.args)
			cmd.Execute()

			t.CheckDeepEqual(test.expected, flag.value)
		})
	}
}

func TestIntOrUndefinedUsage(t *testing.T) {
	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.Flags().Var(&IntOrUndefined{}, "int-flag", "use it like this")
	cmd.SetOut(&output)
	cmd.Usage()
	testutil.CheckDeepEqual(t, "Usage:\n\nFlags:\n      --int-flag int   use it like this\n", output.String())
}

func TestIntOrUndefined_SetNil(t *testing.T) {
	var s IntOrUndefined
	s.Set("1")
	testutil.CheckDeepEqual(t, "1", s.String())
	s.SetNil()
	testutil.CheckDeepEqual(t, "", s.String())
	testutil.CheckDeepEqual(t, (*int)(nil), s.value)
	testutil.CheckDeepEqual(t, (*int)(nil), s.Value())
}
func TestIntOrUndefined(t *testing.T) {
	tests := []struct {
		description string
		args        []string
		expected    *int
	}{
		{
			description: "undefined",
			args:        []string{},
			expected:    nil,
		},
		{
			description: "empty",
			args:        []string{"--int-flag="},
			expected:    nil,
		},
		{
			description: "invalid",
			args:        []string{"--int-flag=invalid"},
			expected:    nil,
		},
		{
			description: "0",
			args:        []string{"--int-flag=0"},
			expected:    util.Ptr(0),
		},
		{
			description: "1",
			args:        []string{"--int-flag=1"},
			expected:    util.Ptr(1),
		},
		{
			description: "-1",
			args:        []string{"--int-flag=-1"},
			expected:    util.Ptr(-1),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var flag IntOrUndefined

			cmd := &cobra.Command{}
			cmd.Flags().Var(&flag, "int-flag", "")
			cmd.SetArgs(test.args)
			cmd.Execute()

			t.CheckDeepEqual(test.expected, flag.value)
		})
	}
}

func TestMuted(t *testing.T) {
	tests := []struct {
		phases                  []string
		expectedMuteBuild       bool
		expectedMuteTest        bool
		expectedMuteStatusCheck bool
		expectedMuteDeploy      bool
	}{
		{
			phases:                  nil,
			expectedMuteBuild:       false,
			expectedMuteTest:        false,
			expectedMuteStatusCheck: false,
			expectedMuteDeploy:      false,
		},
		{
			phases:                  []string{"build"},
			expectedMuteBuild:       true,
			expectedMuteTest:        false,
			expectedMuteStatusCheck: false,
			expectedMuteDeploy:      false,
		},
		{
			phases:                  []string{"test"},
			expectedMuteBuild:       false,
			expectedMuteTest:        true,
			expectedMuteStatusCheck: false,
			expectedMuteDeploy:      false,
		},
		{
			phases:                  []string{"status-check"},
			expectedMuteBuild:       false,
			expectedMuteTest:        false,
			expectedMuteStatusCheck: true,
			expectedMuteDeploy:      false,
		},
		{
			phases:                  []string{"deploy"},
			expectedMuteBuild:       false,
			expectedMuteTest:        false,
			expectedMuteStatusCheck: false,
			expectedMuteDeploy:      true,
		},
		{
			phases:                  []string{"build", "test", "status-check", "deploy"},
			expectedMuteBuild:       true,
			expectedMuteTest:        true,
			expectedMuteStatusCheck: true,
			expectedMuteDeploy:      true,
		},
		{
			phases:                  []string{"all"},
			expectedMuteBuild:       true,
			expectedMuteTest:        true,
			expectedMuteStatusCheck: true,
			expectedMuteDeploy:      true,
		},
		{
			phases:                  []string{"none"},
			expectedMuteBuild:       false,
			expectedMuteTest:        false,
			expectedMuteStatusCheck: false,
			expectedMuteDeploy:      false,
		},
	}
	for _, test := range tests {
		description := strings.Join(test.phases, ",")

		testutil.Run(t, description, func(t *testutil.T) {
			m := Muted{
				Phases: test.phases,
			}

			t.CheckDeepEqual(test.expectedMuteBuild, m.MuteBuild())
			t.CheckDeepEqual(test.expectedMuteTest, m.MuteTest())
			t.CheckDeepEqual(test.expectedMuteStatusCheck, m.MuteStatusCheck())
			t.CheckDeepEqual(test.expectedMuteDeploy, m.MuteDeploy())
		})
	}
}
