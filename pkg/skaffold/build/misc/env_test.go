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

package misc

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEvaluateEnv(t *testing.T) {
	tests := []struct {
		description string
		env         []string
		want        []string
		shouldErr   bool
	}{
		{
			description: "env variables",
			env:         []string{"key1=value1", "key2=", "key3={{.FOO}}"},
			want:        []string{"key1=value1", "key2=", "key3=foo"},
		},
		{
			description: "invalid env",
			env:         []string{"invalid"},
			shouldErr:   true,
		},
		{
			description: "invalid template",
			env:         []string{"key3={{INVALID}}"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=foo", "BAR=bar"} })

			result, err := EvaluateEnv(test.env)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.want, result)
			}
		})
	}
}
