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

package initializer

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintAnalyzeJSON(t *testing.T) {
	tests := []struct {
		name        string
		dockerfiles []string
		images      []string
		skipBuild   bool
		shouldErr   bool
		expected    string
	}{
		{
			name:        "dockerfile and image",
			dockerfiles: []string{"Dockerfile", "Dockerfile_2"},
			images:      []string{"image1", "image2"},
			expected:    "{\"dockerfiles\":[\"Dockerfile\",\"Dockerfile_2\"],\"images\":[\"image1\",\"image2\"]}",
		},
		{
			name:      "no dockerfile, skip build",
			images:    []string{"image1", "image2"},
			skipBuild: true,
			expected:  "{\"images\":[\"image1\",\"image2\"]}"},
		{
			name:      "no dockerfile",
			images:    []string{"image1", "image2"},
			shouldErr: true,
		},
		{
			name:      "no dockerfiles or images",
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := bytes.NewBuffer([]byte{})
			err := printAnalyzeJSON(out, test.skipBuild, test.dockerfiles, test.images)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, out.String())
		})
	}
}
