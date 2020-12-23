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

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestStripTags(t *testing.T) {
	tests := []struct {
		name           string
		images         []string
		expectedImages []string
	}{
		{
			name:           "latest",
			images:         []string{"gcr.io/foo/bar:latest"},
			expectedImages: []string{"gcr.io/foo/bar"},
		},
		{
			name:           "no default repo",
			images:         []string{"foo:bar"},
			expectedImages: []string{"foo"},
		},
		{
			name:           "two images, one without a repo",
			images:         []string{"gcr.io/foo/bar:latest", "foo:bar"},
			expectedImages: []string{"gcr.io/foo/bar", "foo"},
		},
		{
			name:           "ignore digest",
			images:         []string{"foo:sha256@deadbeef"},
			expectedImages: nil,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Parallel()

			i := StripTags(test.images)
			t.CheckDeepEqual(test.expectedImages, i)
		})
	}
}

func TestSetImageTag(t *testing.T) {
	tests := []struct {
		description   string
		image         string
		tag           string
		expectedImage string
		shouldErr     bool
	}{
		{
			description:   "image with tag",
			image:         "gcr.io/foo/bar:latest",
			tag:           "test-1",
			expectedImage: "gcr.io/foo/bar:test-1",
		},
		{
			description:   "image with tag and digest",
			image:         "gcr.io/foo/bar:latest@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc",
			tag:           "test-2",
			expectedImage: "gcr.io/foo/bar:test-2@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc",
		},
		{
			description:   "image without tag and digest",
			image:         "gcr.io/foo/bar",
			tag:           "test-3",
			expectedImage: "gcr.io/foo/bar:test-3",
		},
		{
			description:   "empty tag",
			image:         "gcr.io/foo/bar:test-4",
			expectedImage: "gcr.io/foo/bar",
		},
		{
			description:   "image with digest",
			image:         "gcr.io/foo/bar@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc",
			tag:           "test-5",
			expectedImage: "gcr.io/foo/bar:test-5@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc",
		},
		{
			description: "invalid reference",
			image:       "!!invalid!!",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Parallel()

			image, err := SetImageTag(test.image, test.tag)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedImage, image)
		})
	}
}
