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

package jib

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependenciesMaven(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"mvn jib:_skaffold-files -q",
		"dep1\ndep2\n\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("pom.xml", "")

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesMavenNoPomXml(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(
		"ignored",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	_, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, false)

	if err.Error() != "no pom.xml found" {
		t.Errorf("Unexpected error message %s", err.Error())
	}
}

func TestGetDependenciesGradle(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"gradle _jibSkaffoldFiles -q",
		"dep1\ndep2\n\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("build.gradle", "")

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesGradleNoPomXml(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(
		"ignored",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	_, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, false)

	if err.Error() != "no build.gradle found" {
		t.Errorf("Unexpected error message %s", err.Error())
	}
}

func TestGetCommandMaven(t *testing.T) {
	var tests = []struct {
		description        string
		jibMavenArtifact   v1alpha3.JibMavenArtifact
		filesInWorkspace   []string
		isWindows          bool
		expectedExecutable string
		expectedSubCommand []string
	}{
		{
			description:        "maven default",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{},
			filesInWorkspace:   []string{},
			isWindows:          false,
			expectedExecutable: "mvn",
			expectedSubCommand: []string{"jib:_skaffold-files", "-q"},
		},
		{
			description:        "maven with profile",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace:   []string{},
			isWindows:          false,
			expectedExecutable: "mvn",
			expectedSubCommand: []string{"jib:_skaffold-files", "-q", "-P", "profile"},
		},
		{
			description:        "maven with wrapper",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{},
			filesInWorkspace:   []string{"mvnw"},
			isWindows:          false,
			expectedExecutable: "./mvnw",
			expectedSubCommand: []string{"jib:_skaffold-files", "-q"},
		},
		{
			description:        "maven no wrapper on windows",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{},
			filesInWorkspace:   []string{},
			isWindows:          true,
			expectedExecutable: "mvn",
			expectedSubCommand: []string{"jib:_skaffold-files", "-q"},
		},
		{
			description:        "maven with wrapper on windows",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{},
			filesInWorkspace:   []string{"mvnw.cmd"},
			isWindows:          true,
			expectedExecutable: "cmd.exe",
			expectedSubCommand: []string{"/C", "mvnw.cmd", "jib:_skaffold-files", "-q"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			for _, file := range test.filesInWorkspace {
				tmpDir.Write(file, "")
			}

			executable, subCommand := getCommandMaven(tmpDir.Root(), &test.jibMavenArtifact, test.isWindows)

			if executable != test.expectedExecutable {
				t.Errorf("Expected executable %s. Got %s", test.expectedExecutable, executable)
			}
			testutil.CheckDeepEqual(t, test.expectedSubCommand, subCommand)
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	var tests = []struct {
		description        string
		jibGradleArtifact  v1alpha3.JibGradleArtifact
		filesInWorkspace   []string
		isWindows          bool
		expectedExecutable string
		expectedSubCommand []string
	}{
		{
			description:        "gradle default",
			jibGradleArtifact:  v1alpha3.JibGradleArtifact{},
			filesInWorkspace:   []string{},
			isWindows:          false,
			expectedExecutable: "gradle",
			expectedSubCommand: []string{"_jibSkaffoldFiles", "-q"},
		},
		{
			description:        "gradle with wrapper",
			jibGradleArtifact:  v1alpha3.JibGradleArtifact{},
			filesInWorkspace:   []string{"gradlew"},
			isWindows:          false,
			expectedExecutable: "./gradlew",
			expectedSubCommand: []string{"_jibSkaffoldFiles", "-q"},
		},
		{
			description:        "gradle no wrapper on windows",
			jibGradleArtifact:  v1alpha3.JibGradleArtifact{},
			filesInWorkspace:   []string{},
			isWindows:          true,
			expectedExecutable: "gradle",
			expectedSubCommand: []string{"_jibSkaffoldFiles", "-q"},
		},
		{
			description:        "gradle with wrapper on windows",
			jibGradleArtifact:  v1alpha3.JibGradleArtifact{},
			filesInWorkspace:   []string{"gradlew.bat"},
			isWindows:          true,
			expectedExecutable: "cmd.exe",
			expectedSubCommand: []string{"/C", "gradlew.bat", "_jibSkaffoldFiles", "-q"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			for _, file := range test.filesInWorkspace {
				tmpDir.Write(file, "")
			}

			executable, subCommand := getCommandGradle(tmpDir.Root(), &test.jibGradleArtifact, test.isWindows)

			if executable != test.expectedExecutable {
				t.Errorf("Expected executable %s. Got %s", test.expectedExecutable, executable)
			}
			testutil.CheckDeepEqual(t, test.expectedSubCommand, subCommand)
		})
	}
}

func TestGetDepsFromStdout(t *testing.T) {
	var tests = []struct {
		stdout       string
		expectedDeps []string
	}{
		{
			stdout:       "",
			expectedDeps: []string{},
		},
		{
			stdout:       "deps1\ndeps2",
			expectedDeps: []string{"deps1", "deps2"},
		},
		{
			stdout:       "deps1\ndeps2\n",
			expectedDeps: []string{"deps1", "deps2"},
		},
		{
			stdout:       "\n\n\n",
			expectedDeps: []string{},
		},
		{
			stdout:       "\n\ndeps1\n\ndeps2\n\n\n",
			expectedDeps: []string{"deps1", "deps2"},
		},
	}

	for _, test := range tests {
		t.Run("getDepsFromStdout", func(t *testing.T) {
			deps := getDepsFromStdout(test.stdout)
			testutil.CheckDeepEqual(t, test.expectedDeps, deps)
		})
	}
}
