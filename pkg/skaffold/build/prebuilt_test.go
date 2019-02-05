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

package build

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPreBuiltImagesBuilder(t *testing.T) {
	var tests = []struct {
		description string
		images      []string
		artifacts   []*latest.Artifact
		expected    []Artifact
		shouldErr   bool
	}{
		{
			description: "images in same order",
			images: []string{
				"skaffold/image1:tag1",
				"skaffold/image2:tag2",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
				{ImageName: "skaffold/image2", Tag: "skaffold/image2:tag2"},
			},
		},
		{
			description: "images in reverse order",
			images: []string{
				"skaffold/image2:tag2",
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
				{ImageName: "skaffold/image2", Tag: "skaffold/image2:tag2"},
			},
		},
		{
			description: "missing image",
			images: []string{
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			shouldErr: true,
		},
		{
			// Should we support that? It is used in kustomize example.
			description: "additional image",
			images: []string{
				"busybox:1",
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
			},
			expected: []Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
				{ImageName: "busybox", Tag: "busybox:1"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			builder := NewPreBuiltImagesBuilder(test.images)

			bRes, err := builder.Build(context.Background(), ioutil.Discard, nil, test.artifacts)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, bRes)
		})
	}
}
