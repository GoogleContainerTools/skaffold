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

package cmd

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDoDeploy(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []build.Artifact
		expected    bool
		shouldErr   bool
	}{
		{
			description: "no artifact",
			artifacts:   nil,
			expected:    true,
			shouldErr:   false,
		},
		{
			description: "missing tags",
			artifacts:   []build.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			expected:    false,
			shouldErr:   true,
		},
		{
			description: "one missing tag",
			artifacts:   []build.Artifact{{ImageName: "image1"}, {ImageName: "image2", Tag: "image2:tag"}},
			expected:    false,
			shouldErr:   true,
		},
		{
			description: "with tags",
			artifacts:   []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}, {ImageName: "image2", Tag: "image2:tag"}},
			expected:    true,
			shouldErr:   false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deployed, err := validateArtifactTags(test.artifacts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deployed)
		})
	}
}
