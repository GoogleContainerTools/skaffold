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

func TestValidate(t *testing.T) {
	var tests = []struct {
		description   string
		path          string
		expectedValid bool
	}{
		{
			description:   "NodeJS",
			path:          filepath.Join("path", "to", "package.json"),
			expectedValid: true,
		},
		{
			description:   "NodeJS (root)",
			path:          filepath.Join("package.json"),
			expectedValid: true,
		},
		{
			description:   "Go",
			path:          filepath.Join("path", "to", "go.mod"),
			expectedValid: true,
		},
		{
			description:   "Go (root)",
			path:          filepath.Join("go.mod"),
			expectedValid: true,
		},
		{
			description:   "Buildpacks",
			path:          filepath.Join("project.toml"),
			expectedValid: true,
		},
		{
			description:   "Unknown language",
			path:          filepath.Join("path", "to", "something.txt"),
			expectedValid: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Touch(test.path)

			isValid := Validate(tmpDir.Path(test.path))

			t.CheckDeepEqual(test.expectedValid, isValid)
		})
	}
}

func TestValidateIgnored(t *testing.T) {
	paths := []string{
		filepath.Join("parent", "node_modules", "package.json"),
		filepath.Join("node_modules", "package.json"),
		filepath.Join("vendor", "go.mod"),
		filepath.Join("parent", "vendor", "go.mod"),
		filepath.Join("parent", "vendor", "project.toml"),
		filepath.Join("node_modules", "project.toml"),
	}

	for _, path := range paths {
		testutil.Run(t, path, func(t *testutil.T) {
			isValid := Validate(path)

			t.CheckFalse(isValid)
		})
	}
}

func TestDescribe(t *testing.T) {
	var tests = []struct {
		description    string
		config         ArtifactConfig
		expectedPrompt string
	}{
		{
			description:    "buildpacks - NodeJS",
			config:         ArtifactConfig{File: "/path/to/package.json"},
			expectedPrompt: "Buildpacks (/path/to/package.json)",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expectedPrompt, test.config.Describe())
		})
	}
}

func TestArtifactType(t *testing.T) {
	var tests = []struct {
		description  string
		config       ArtifactConfig
		expectedType latest.ArtifactType
	}{
		{
			description: "buildpacks - NodeJS",
			config: ArtifactConfig{
				File:    filepath.Join("path", "to", "package.json"),
				Builder: "some/builder",
			},
			expectedType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{
					Builder: "some/builder",
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
