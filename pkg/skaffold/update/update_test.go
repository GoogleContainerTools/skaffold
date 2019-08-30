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

package update

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsUpdateCheckEnabledByEnvOrConfig(t *testing.T) {
	tests := []struct {
		description string
		envVariable string
		configCheck bool
		expected    bool
	}{
		{
			description: "env variable is set to true",
			envVariable: "true",
			expected:    true,
		},
		{
			description: "env variable is set to false",
			envVariable: "false",
		},
		{
			description: "env variable is set to random string",
			envVariable: "foo",
		},
		{
			description: "env variable is empty and config is enabled",
			configCheck: true,
			expected:    true,
		},
		{
			description: "env variable is false but Global update-check config is true",
			envVariable: "false",
			configCheck: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&isConfigUpdateCheckEnabled, func(string) bool { return test.configCheck })
			t.Override(&getEnv, func(string) string { return test.envVariable })
			t.CheckDeepEqual(test.expected, isUpdateCheckEnabledByEnvOrConfig("dummyconfig"))
		})
	}
}
