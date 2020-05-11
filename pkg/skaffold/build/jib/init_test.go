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

func TestValidate(t *testing.T) {
	var tests = []struct {
		description    string
		path           string
		enableGradle   bool
		fileContents   string
		command        string
		stdout         string
		expectedConfig []ArtifactConfig
	}{
		{
			description:    "not a jib file",
			path:           "path/to/something.txt",
			enableGradle:   true,
			expectedConfig: nil,
		},
		{
			description:    "jib string not found",
			path:           "path/to/build.gradle",
			enableGradle:   true,
			fileContents:   "not a useful string",
			expectedConfig: nil,
		},
		{
			description:    "jib string found but not configured",
			path:           "path/to/build.gradle",
			enableGradle:   true,
			fileContents:   "com.google.cloud.tools.jib",
			command:        "gradle _jibSkaffoldInit -q --console=plain",
			stdout:         "error",
			expectedConfig: nil,
		},
		{
			description:  "jib gradle single project",
			path:         "path/to/build.gradle",
			enableGradle: true,
			fileContents: "com.google.cloud.tools.jib",
			command:      "gradle _jibSkaffoldInit -q --console=plain",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project"}
`,
			expectedConfig: []ArtifactConfig{
				{BuilderName: PluginName(JibGradle), File: "path/to/build.gradle", Image: "image", Project: "project"},
			},
		},
		{
			description:  "jib gradle-kotlin single project",
			path:         "path/to/build.gradle.kts",
			enableGradle: true,
			fileContents: "com.google.cloud.tools.jib",
			command:      "gradle _jibSkaffoldInit -q --console=plain",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project"}
`,
			expectedConfig: []ArtifactConfig{
				{BuilderName: PluginName(JibGradle), File: "path/to/build.gradle.kts", Image: "image", Project: "project"},
			},
		},
		{
			description:  "jib gradle multi-project",
			enableGradle: true,
			path:         "path/to/build.gradle",
			fileContents: "com.google.cloud.tools.jib",
			command:      "gradle _jibSkaffoldInit -q --console=plain",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project1"}

BEGIN JIB JSON
{"project":"project2"}
`,
			expectedConfig: []ArtifactConfig{
				{BuilderName: PluginName(JibGradle), File: "path/to/build.gradle", Image: "image", Project: "project1"},
				{BuilderName: PluginName(JibGradle), File: "path/to/build.gradle", Project: "project2"},
			},
		},
		{
			description:    "jib gradle disabled",
			path:           "path/to/build.gradle",
			enableGradle:   false,
			fileContents:   "com.google.cloud.tools.jib",
			command:        "",
			stdout:         ``,
			expectedConfig: nil,
		},
		{
			description:  "jib maven single module",
			path:         "path/to/pom.xml",
			enableGradle: true,
			fileContents: "<artifactId>jib-maven-plugin</artifactId>",
			command:      "mvn jib:_skaffold-init -q --batch-mode",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project"}`,
			expectedConfig: []ArtifactConfig{
				{BuilderName: PluginName(JibMaven), File: "path/to/pom.xml", Image: "image", Project: "project"},
			},
		},
		{
			description:  "jib maven multi-module",
			path:         "path/to/pom.xml",
			fileContents: "<artifactId>jib-maven-plugin</artifactId>",
			command:      "mvn jib:_skaffold-init -q --batch-mode",
			stdout: `BEGIN JIB JSON
{"image":"image","project":"project1"}

BEGIN JIB JSON
{"project":"project2"}
`,
			expectedConfig: []ArtifactConfig{
				{BuilderName: PluginName(JibMaven), File: "path/to/pom.xml", Image: "image", Project: "project1"},
				{BuilderName: PluginName(JibMaven), File: "path/to/pom.xml", Project: "project2"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Write(test.path, test.fileContents)
			for i := range test.expectedConfig {
				test.expectedConfig[i].File = tmpDir.Path(test.expectedConfig[i].File)
			}

			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.command,
				test.stdout,
			))

			validated := Validate(tmpDir.Path(test.path), test.enableGradle)

			t.CheckDeepEqual(test.expectedConfig, validated)
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
			description:    "gradle without project",
			config:         ArtifactConfig{BuilderName: PluginName(JibGradle), File: "path/to/build.gradle"},
			expectedPrompt: "Jib Gradle Plugin (path/to/build.gradle)",
		},
		{
			description:    "gradle with project",
			config:         ArtifactConfig{BuilderName: PluginName(JibGradle), Project: "project", File: "path/to/build.gradle"},
			expectedPrompt: "Jib Gradle Plugin (project, path/to/build.gradle)",
		},
		{
			description:    "maven without project",
			config:         ArtifactConfig{BuilderName: PluginName(JibMaven), File: "path/to/pom.xml"},
			expectedPrompt: "Jib Maven Plugin (path/to/pom.xml)",
		},
		{
			description:    "maven with project",
			config:         ArtifactConfig{BuilderName: PluginName(JibMaven), Project: "project", File: "path/to/pom.xml"},
			expectedPrompt: "Jib Maven Plugin (project, path/to/pom.xml)",
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
			description:  "jib gradle",
			config:       ArtifactConfig{BuilderName: "Jib Gradle Plugin", File: filepath.Join("path", "to", "build.gradle"), Project: "project"},
			expectedType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{Project: "project"}},
		},
		{
			description:  "jib gradle without project",
			config:       ArtifactConfig{BuilderName: "Jib Gradle Plugin", File: filepath.Join("path", "to", "build.gradle")},
			expectedType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}},
		},
		{
			description:  "jib maven",
			config:       ArtifactConfig{BuilderName: "Jib Maven Plugin", File: filepath.Join("path", "to", "pom.xml"), Project: "project"},
			expectedType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{Project: "project"}},
		},
		{
			description:  "jib maven without project",
			config:       ArtifactConfig{BuilderName: "Jib Maven Plugin", File: filepath.Join("path", "to", "pom.xml")},
			expectedType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			at := test.config.ArtifactType()

			t.CheckDeepEqual(test.expectedType, at)
		})
	}
}
