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
	"testing"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestStringOrUndefinedUsage(t *testing.T) {
	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.Flags().Var(&StringOrUndefined{}, "flag", "use it like this")
	cmd.SetOutput(&output)
	cmd.Usage()

	testutil.CheckDeepEqual(t, "Usage:\n\nFlags:\n      --flag string   use it like this\n", output.String())
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
			expected:    util.StringPtr("value"),
		},
		{
			description: "empty",
			args:        []string{"--flag="},
			expected:    util.StringPtr(""),
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
