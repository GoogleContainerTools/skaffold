/*
Copyright 2022 The Skaffold Authors

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

package kaniko

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetContext(t *testing.T) {
	tests := []struct {
		description string
		context     string
		artifact    *latest.KanikoArtifact
		expected    string
	}{
		{
			description: "with ContextSubPath",
			context:     "dir/",
			artifact: &latest.KanikoArtifact{
				ContextSubPath: "sub/",
			},
			expected: filepath.FromSlash("dir/sub"),
		},
		{
			description: "without ContextSubPath",
			context:     "dir/",
			artifact:    &latest.KanikoArtifact{},
			expected:    filepath.FromSlash("dir"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := GetContext(test.artifact, test.context)
			t.CheckDeepEqual(got, test.expected)
		})
	}
}
