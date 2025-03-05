/*
Copyright 2025 The Skaffold Authors

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

package helm

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildDepBuildArgs(t *testing.T) {
	tests := []struct {
		description                  string
		skipBuildDependenciesRefresh bool
		expected                     []string
	}{
		{
			description:                  "build args without skipBuildDependenciesRefresh",
			skipBuildDependenciesRefresh: false,
			expected:                     []string{"dep", "build"},
		},
		{
			description:                  "build args with skipBuildDependenciesRefresh",
			skipBuildDependenciesRefresh: true,
			expected:                     []string{"dep", "build", "--skip-refresh"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := buildDepBuildArgs(test.skipBuildDependenciesRefresh)
			t.CheckDeepEqual(test.expected, args)
		})
	}
}
