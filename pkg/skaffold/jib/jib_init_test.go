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

package jib

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestValidateJibConfig(t *testing.T) {
	var tests = []struct {
		description    string
		path           string
		command        string
		stdout         string
		expectedConfig []Jib
	}{
		{
			description:    "not a jib file",
			path:           "path/to/something.txt",
			expectedConfig: nil,
		},
		{
			description:    "jib not configured",
			path:           "path/to/build.gradle",
			command:        "gradle _jibSkaffoldInit -q",
			stdout:         "error",
			expectedConfig: nil,
		},
		{
			description: "jib gradle single project",
			path:        "path/to/build.gradle",
			command:     "gradle _jibSkaffoldInit -q",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project"}
`,
			expectedConfig: []Jib{
				{BuilderName: JibGradle.Name(), FilePath: "path/to/build.gradle", Image: "image", Project: "project"},
			},
		},
		{
			description: "jib gradle multi-project",
			path:        "path/to/build.gradle",
			command:     "gradle _jibSkaffoldInit -q",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project1"}

BEGIN JIB JSON
{"project":"project2"}
`,
			expectedConfig: []Jib{
				{BuilderName: JibGradle.Name(), FilePath: "path/to/build.gradle", Image: "image", Project: "project1"},
				{BuilderName: JibGradle.Name(), FilePath: "path/to/build.gradle", Project: "project2"},
			},
		},
		{
			description: "jib maven single module",
			path:        "path/to/pom.xml",
			command:     "mvn jib:_skaffold-init -q",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project"}`,
			expectedConfig: []Jib{
				{BuilderName: JibMaven.Name(), FilePath: "path/to/pom.xml", Image: "image", Project: "project"},
			},
		},
		{
			description: "jib maven multi-module",
			path:        "path/to/pom.xml",
			command:     "mvn jib:_skaffold-init -q",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project1"}

BEGIN JIB JSON
{"project":"project2"}
`,
			expectedConfig: []Jib{
				{BuilderName: JibMaven.Name(), FilePath: "path/to/pom.xml", Image: "image", Project: "project1"},
				{BuilderName: JibMaven.Name(), FilePath: "path/to/pom.xml", Project: "project2"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.command,
				test.stdout,
			))

			validated := ValidateJibConfig(test.path)

			t.CheckDeepEqual(test.expectedConfig, validated)
		})
	}
}

func TestDescribe(t *testing.T) {
	var tests = []struct {
		description    string
		config         Jib
		expectedPrompt string
	}{
		{
			description:    "gradle without project",
			config:         Jib{BuilderName: JibGradle.Name(), FilePath: "path/to/build.gradle"},
			expectedPrompt: "Jib Gradle Plugin (path/to/build.gradle)",
		},
		{
			description:    "gradle with project",
			config:         Jib{BuilderName: JibGradle.Name(), Project: "project", FilePath: "path/to/build.gradle"},
			expectedPrompt: "Jib Gradle Plugin (project, path/to/build.gradle)",
		},
		{
			description:    "maven without project",
			config:         Jib{BuilderName: JibMaven.Name(), FilePath: "path/to/pom.xml"},
			expectedPrompt: "Jib Maven Plugin (path/to/pom.xml)",
		},
		{
			description:    "maven with project",
			config:         Jib{BuilderName: JibMaven.Name(), Project: "project", FilePath: "path/to/pom.xml"},
			expectedPrompt: "Jib Maven Plugin (project, path/to/pom.xml)",
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
		config           Jib
		manifestImage    string
		expectedArtifact latest.Artifact
		expectedImage    string
	}{
		{
			description:   "jib gradle with image and project",
			config:        Jib{BuilderName: JibGradle.Name(), FilePath: filepath.Join("path", "to", "build.gradle"), Image: "image", Project: "project"},
			manifestImage: "different-image",
			expectedArtifact: latest.Artifact{
				ImageName: "image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{
					JibGradleArtifact: &latest.JibGradleArtifact{Project: "project"},
				},
			},
		},
		{
			description:   "jib gradle without image and project",
			config:        Jib{BuilderName: JibGradle.Name(), FilePath: filepath.Join("path", "to", "build.gradle")},
			manifestImage: "different-image",
			expectedArtifact: latest.Artifact{
				ImageName: "different-image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{
					JibGradleArtifact: &latest.JibGradleArtifact{
						Flags: []string{"-Dimage=different-image"},
					},
				},
			},
		},
		{
			description:   "jib maven with image and project",
			config:        Jib{BuilderName: JibMaven.Name(), FilePath: filepath.Join("path", "to", "pom.xml"), Image: "image", Project: "project"},
			manifestImage: "different-image",
			expectedArtifact: latest.Artifact{
				ImageName: "image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{Module: "project"},
				},
			},
		},
		{
			description:   "jib maven without image and project",
			config:        Jib{BuilderName: JibMaven.Name(), FilePath: filepath.Join("path", "to", "pom.xml")},
			manifestImage: "different-image",
			expectedArtifact: latest.Artifact{
				ImageName: "different-image",
				Workspace: filepath.Join("path", "to"),
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{
						Flags: []string{"-Dimage=different-image"},
					},
				},
			},
		},
		{
			description:   "ignore workspace",
			config:        Jib{BuilderName: JibGradle.Name(), FilePath: "build.gradle", Image: "image"},
			manifestImage: "different-image",
			expectedArtifact: latest.Artifact{
				ImageName:    "image",
				ArtifactType: latest.ArtifactType{JibGradleArtifact: &latest.JibGradleArtifact{}},
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
