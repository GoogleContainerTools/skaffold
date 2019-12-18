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

package buildpacks

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestValidateConfig(t *testing.T) {
	var tests = []struct {
		description   string
		path          string
		expectedValid bool
	}{
		{
			description:   "NodeJS",
			path:          "path/to/package.json",
			expectedValid: true,
		},
		{
			description:   "Unknown language",
			path:          "path/to/something.txt",
			expectedValid: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch(test.path)

			isValid := ValidateConfig(tmpDir.Path(test.path))

			t.CheckDeepEqual(test.expectedValid, isValid)
		})
	}
}

func TestDescribe(t *testing.T) {
	var tests = []struct {
		description    string
		config         Buildpacks
		expectedPrompt string
	}{
		{
			description:    "buildpacks",
			config:         Buildpacks{File: "/path/to/package.json"},
			expectedPrompt: "Buildpacks (/path/to/package.json)",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expectedPrompt, test.config.Describe())
		})
	}
}

func TestCreateArtifact(t *testing.T) {
	var tests = []struct {
		description      string
		config           Buildpacks
		manifestImage    string
		expectedArtifact latest.Artifact
		expectedImage    string
	}{
		{
			description:   "buildpacks",
			config:        Buildpacks{File: filepath.Join("path", "to", "package.json")},
			manifestImage: "image",
			expectedArtifact: latest.Artifact{
				ImageName: "image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{BuildpackArtifact: &latest.BuildpackArtifact{
					Builder: "heroku/buildpacks",
				}},
			},
		},
		{
			description:   "ignore workspace",
			config:        Buildpacks{File: "build.gradle"},
			manifestImage: "other-image",
			expectedArtifact: latest.Artifact{
				ImageName: "other-image",
				ArtifactType: latest.ArtifactType{BuildpackArtifact: &latest.BuildpackArtifact{
					Builder: "heroku/buildpacks",
				}},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := test.config.CreateArtifact(test.manifestImage)

			t.CheckDeepEqual(test.expectedArtifact, *artifact)
		})
	}
}
