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

func TestIsUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		description string
		enabled     bool
		configCheck bool
		expected    bool
	}{
		{
			description: "globally disabled - disabled in config -> disabled",
			enabled:     false,
			configCheck: false,
			expected:    false,
		},
		{
			description: "globally enabled - disabled in config -> disabled",
			enabled:     true,
			configCheck: false,
			expected:    false,
		},
		{
			description: "globally disabled - enabled in config -> disabled",
			enabled:     false,
			configCheck: true,
			expected:    false,
		},
		{
			description: "globally enabled - enabled in config -> enabled",
			enabled:     true,
			configCheck: true,
			expected:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&EnableCheck, test.enabled)
			t.Override(&isConfigUpdateCheckEnabled, func(string) bool { return test.configCheck })

			isEnabled := IsUpdateCheckEnabled("dummyconfig")

			t.CheckDeepEqual(test.expected, isEnabled)
		})
	}
}
