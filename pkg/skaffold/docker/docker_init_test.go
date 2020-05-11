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

package docker

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		description    string
		content        string
		fileToValidate string
		expectedValid  bool
	}{
		{
			description:    "valid",
			content:        "FROM scratch",
			fileToValidate: "Dockerfile",
			expectedValid:  true,
		},
		{
			description:    "invalid command",
			content:        "GARBAGE",
			fileToValidate: "Dockerfile",
			expectedValid:  false,
		},
		{
			description:    "not found",
			fileToValidate: "Unknown",
			expectedValid:  false,
		},
		{
			description:    "invalid file",
			content:        "#escape",
			fileToValidate: "Dockerfile",
			expectedValid:  false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("Dockerfile", test.content)

			valid := Validate(tmpDir.Path(test.fileToValidate))

			t.CheckDeepEqual(test.expectedValid, valid)
		})
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		description    string
		dockerfile     ArtifactConfig
		expectedPrompt string
	}{
		{
			description:    "Dockerfile prompt",
			dockerfile:     ArtifactConfig{File: "path/to/Dockerfile"},
			expectedPrompt: "Docker (path/to/Dockerfile)",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expectedPrompt, test.dockerfile.Describe())
		})
	}
}

func TestArtifactType(t *testing.T) {
	tests := []struct {
		description  string
		config       ArtifactConfig
		expectedType latest.ArtifactType
	}{
		{
			description:  "default filename",
			config:       ArtifactConfig{File: filepath.Join("path", "to", "Dockerfile")},
			expectedType: latest.ArtifactType{},
		},
		{
			description: "non-default filename",
			config:      ArtifactConfig{File: filepath.Join("path", "to", "Dockerfile1")},
			expectedType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: "Dockerfile1",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			at := test.config.ArtifactType()

			t.CheckDeepEqual(test.expectedType, at)
		})
	}
}
