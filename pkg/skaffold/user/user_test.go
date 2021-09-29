/*
Copyright 2021 The Skaffold Authors

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

package user

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestAllowedUser(t *testing.T) {
	tests := []struct {
		description string
		user        string
		expected    bool
	}{
		{
			description: "user in allowed list",
			user:        "vsc",
			expected:    true,
		},
		{
			description: "user in allowed list",
			user:        "cloud-deploy",
			expected:    true,
		},
		{
			description: "user in allowed list with valid pattern",
			user:        "cloud-deploy/dev",
			expected:    true,
		},
		{
			description: "user in allowed list with invalid pattern (only slash)",
			user:        "cloud-deploy|dev",
			expected:    false,
		},
		{
			description: "user in allowed list with invalid pattern (suffix required)",
			user:        "cloud-deploy/",
			expected:    false,
		},
		{
			description: "user not in allowed list",
			user:        "test-user",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			allowedUser := IsAllowedUser(test.user)
			t.CheckDeepEqual(test.expected, allowedUser)
		})
	}
}
