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

package build

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestSanitizeImageName(t *testing.T) {
	tests := []struct {
		description string
		imageName   string
		expected    string
	}{
		{
			description: "Already Good",
			imageName:   "image-name",
			expected:    "image-name",
		},
		{
			description: "Needs Fixing Slashes",
			imageName:   "image/name/slash",
			expected:    "image-name-slash",
		},
		{
			description: "Needs Fixing Periods",
			imageName:   "image.name.period",
			expected:    "image-name-period",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			sanitized := sanitizeImageName(test.imageName)

			t.CheckDeepEqual(test.expected, sanitized)
		})
	}
}
