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

func TestValidateDockerfile(t *testing.T) {
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

			valid := ValidateDockerfile(tmpDir.Path(test.fileToValidate))

			t.CheckDeepEqual(test.expectedValid, valid)
		})
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		description    string
		dockerfile     Docker
		expectedPrompt string
	}{
		{
			description:    "Dockerfile prompt",
			dockerfile:     Docker{File: "path/to/Dockerfile"},
			expectedPrompt: "Docker (path/to/Dockerfile)",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expectedPrompt, test.dockerfile.Describe())
		})
	}
}

func TestCreateArtifact(t *testing.T) {
	tests := []struct {
		description      string
		dockerfile       Docker
		manifestImage    string
		expectedArtifact latest.Artifact
	}{
		{
			description:   "default filename",
			dockerfile:    Docker{File: filepath.Join("path", "to", "Dockerfile")},
			manifestImage: "image",
			expectedArtifact: latest.Artifact{
				ImageName:    "image",
				Workspace:    filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{},
			},
		},
		{
			description:   "non-default filename",
			dockerfile:    Docker{File: filepath.Join("path", "to", "Dockerfile1")},
			manifestImage: "image",
			expectedArtifact: latest.Artifact{
				ImageName: "image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{DockerfilePath: filepath.Join("path", "to", "Dockerfile1")},
				},
			},
		},
		{
			description:   "ignore workspace",
			dockerfile:    Docker{File: "Dockerfile"},
			manifestImage: "image",
			expectedArtifact: latest.Artifact{
				ImageName:    "image",
				ArtifactType: latest.ArtifactType{},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := test.dockerfile.CreateArtifact(test.manifestImage)

			t.CheckDeepEqual(test.expectedArtifact, *artifact)
		})
	}
}
