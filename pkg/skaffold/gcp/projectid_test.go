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

package gcp

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestExtractProjectID(t *testing.T) {
	tests := []struct {
		description string
		imageName   string
		expected    string
		shouldErr   bool
	}{
		{
			description: "gcr.io",
			imageName:   "gcr.io/project/image",
			expected:    "project",
		},
		{
			description: "eu.gcr.io",
			imageName:   "gcr.io/project/image",
			expected:    "project",
		},
		{
			description: "docker hub",
			imageName:   "project/image",
			shouldErr:   true,
		},
		{
			description: "invalid GCR image",
			imageName:   "gcr.io",
			shouldErr:   true,
		},
		{
			description: "invalid reference",
			imageName:   "",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			projectID, err := ExtractProjectID(test.imageName)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, projectID)
		})
	}
}
