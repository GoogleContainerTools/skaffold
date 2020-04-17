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
