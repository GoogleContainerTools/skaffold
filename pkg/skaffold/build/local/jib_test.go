/*
Copyright 2018 The Skaffold Authors

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

package local

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateMavenCommand(t *testing.T) {
	var testCases = []struct {
		in  latest.JibMavenArtifact
		out []string
	}{
		{latest.JibMavenArtifact{}, []string{"prepare-package", "jib:dockerBuild", "-Dimage=image"}},
		{latest.JibMavenArtifact{Profile: "profile"}, []string{"prepare-package", "jib:dockerBuild", "-Dimage=image", "-Pprofile"}},
	}

	for _, tt := range testCases {
		commandLine, err := generateMavenCommand(".", "image", &tt.in)

		testutil.CheckError(t, false, err)
		testutil.CheckDeepEqual(t, tt.out, commandLine)
	}
}

func TestGenerateMavenCommand_errorWithModule(t *testing.T) {
	a := latest.JibMavenArtifact{Module: "module"}
	_, err := generateMavenCommand(".", "image", &a)

	testutil.CheckError(t, true, err)
}

func TestGenerateGradleCommand(t *testing.T) {
	var testCases = []struct {
		in  latest.JibGradleArtifact
		out []string
	}{
		{latest.JibGradleArtifact{}, []string{":jibDockerBuild", "--image=image"}},
		{latest.JibGradleArtifact{Project: "project"}, []string{":project:jibDockerBuild", "--image=image"}},
	}

	for _, tt := range testCases {
		commandLine := generateGradleCommand(".", "image", &tt.in)

		testutil.CheckDeepEqual(t, tt.out, commandLine)
	}
}

func TestGenerateJibImageRef(t *testing.T) {
	var testCases = []struct {
		workspace string
		project   string
		out       string
	}{
		{"simple", "", "jibsimple"},
		{"simple", "project", "jibsimple_project"},
		{".", "project", "jib__d8c7cbe8892fe8442b7f6ef42026769ee6a01e67"},
		{"complex/workspace", "project", "jib__965ec099f720d3ccc9c038c21ea4a598c9632883"},
	}

	for _, tt := range testCases {
		computed := generateJibImageRef(tt.workspace, tt.project)
		if tt.out != computed {
			t.Errorf("Expected '%s' for '%s'/'%s': '%s'", tt.out, tt.workspace, tt.project, computed)
		}
	}
}
