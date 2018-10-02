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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"testing"
)

func TestGenerateMavenCommand(t *testing.T) {
	a := v1alpha3.JibMavenArtifact{}
	expectedSubCommand := []string{"prepare-package", "com.google.cloud.tools:jib-maven-plugin::dockerBuild", "-Dimage=image"}
	commandLine, err := generateMavenCommand(".", "image", &a)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, expectedSubCommand, commandLine)
}

func TestGenerateMavenCommand_withProfile(t *testing.T) {
	a := v1alpha3.JibMavenArtifact{ Profile: "profile"}
	expectedSubCommand := []string{"prepare-package", "com.google.cloud.tools:jib-maven-plugin::dockerBuild", "-Dimage=image", "-Pprofile"}
	commandLine, err := generateMavenCommand(".", "image", &a)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, expectedSubCommand, commandLine)
}

func TestGenerateMavenCommand_withModule(t *testing.T) {
	a := v1alpha3.JibMavenArtifact{ Module: "module"}
	_, err := generateMavenCommand(".", "image", &a)

	testutil.CheckError(t, true, err)
}

func TestGenerateGradleCommand(t *testing.T) {
	a := v1alpha3.JibGradleArtifact{}
	expectedSubCommand := []string{":jibDockerBuild", "--image=image"}
	commandLine, err := generateGradleCommand(".", "image", &a)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, expectedSubCommand, commandLine)
}

func TestGenerateGradleCommand_withProject(t *testing.T) {
	a := v1alpha3.JibGradleArtifact{Project: "project"}
	expectedSubCommand := []string{":project:jibDockerBuild", "--image=image"}
	commandLine, err := generateGradleCommand(".", "image", &a)

	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, expectedSubCommand, commandLine)
}

func TestGenerateJibImageRef_simple_noProject(t *testing.T) {
	imageName := generateJibImageRef("simple", "")
	assertEquals(t, "jibsimple", imageName)
}

func TestGenerateJibImageRef_simple_withProject(t *testing.T) {
	imageName := generateJibImageRef("simple", "project")
	assertEquals(t, "jibsimple_project", imageName)
}

func TestGenerateJibImageRef_complex(t *testing.T) {
	imageName := generateJibImageRef("complex/workspace", "project")
	assertEquals(t, "jib__965ec099f720d3ccc9c038c21ea4a598c9632883", imageName)
}

func assertEquals(t *testing.T, expected, computed string) {
	if expected != computed {
		t.Errorf("Expected '%s': '%s'", expected, computed)
	}
}

