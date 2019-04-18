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

package runner

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPreBuiltImagesBuilder(t *testing.T) {
	var tests = []struct {
		description string
		images      []string
		expected    []build.Artifact
		shouldErr   bool
	}{
		{
			description: "images in same order",
			images: []string{
				"skaffold/image1:tag1",
				"skaffold/image2:tag2",
			},
			expected: []build.Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
				{ImageName: "skaffold/image2", Tag: "skaffold/image2:tag2"},
			},
		},
		{
			// Should we support that? It is used in kustomize example.
			description: "additional image",
			images: []string{
				"busybox:1",
				"skaffold/image1:tag1",
			},
			expected: []build.Artifact{
				{ImageName: "busybox", Tag: "busybox:1"},
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			bRes, err := convertImagesToArtifact(test.images)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, bRes)
		})
	}
}
