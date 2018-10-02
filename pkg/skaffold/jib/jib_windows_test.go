// +build windows

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
	"github.com/GoogleContainerTools/skaffold/testutil"
	"path/filepath"
)

func TestGetCommandMavenWithWrapper(t *testing.T) {
	var tests = []struct {
		description        string
		jibMavenArtifact   v1alpha3.JibMavenArtifact
		filesInWorkspace   []string
		expectedExecutable string
		expectedSubCommand func(workspace string) []string
	}{
		{
			description:        "maven with wrapper",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{},
			filesInWorkspace:   []string{"mvnw.cmd"},
			expectedExecutable: "cmd.exe",
			expectedSubCommand: func(workspace string) []string {
				return []string{"/C", filepath.Join(workspace, "mvnw.cmd"), "jib:_skaffold-files", "-q"}
			},
		},
		{
			description:        "maven with wrapper and profile",
			jibMavenArtifact:   v1alpha3.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace:   []string{"mvnw.cmd"},
			expectedExecutable: "cmd.exe",
			expectedSubCommand: func(workspace string) []string {
				return []string{"/c", filepath.Join(workspace, "mvnw.cmd"), "jib:_skaffold-files", "-q", "-P", "profile"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			for _, file := range test.filesInWorkspace {
				tmpDir.Write(file, "")
			}

			executable, subCommand := getCommandMaven(tmpDir.Root(), &test.jibMavenArtifact)

			if executable != test.expectedExecutable {
				t.Errorf("Expected executable %s. Got %s", test.expectedExecutable, executable)
			}
			testutil.CheckDeepEqual(t, test.expectedSubCommand(tmpDir.Root()), subCommand)
		})
	}
}

func TestGetCommandGradleWithWrapper(t *testing.T) {
	var tests = []struct {
		description        string
		jibGradleArtifact  v1alpha3.JibGradleArtifact
		filesInWorkspace   []string
		expectedExecutable string
		expectedSubCommand func(workspace string) []string
	}{
		{
			description:        "gradle with wrapper",
			jibGradleArtifact:  v1alpha3.JibGradleArtifact{},
			filesInWorkspace:   []string{"gradlew.bat"},
			expectedExecutable: "cmd.exe",
			expectedSubCommand: func(workspace string) []string {
				return []string{"/C", filepath.Join(workspace, "gradlew.bat"), "_jibSkaffoldFiles", "-q"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			for _, file := range test.filesInWorkspace {
				tmpDir.Write(file, "")
			}

			executable, subCommand := getCommandGradle(tmpDir.Root(), &test.jibGradleArtifact)

			if executable != test.expectedExecutable {
				t.Errorf("Expected executable %s. Got %s", test.expectedExecutable, executable)
			}
			testutil.CheckDeepEqual(t, test.expectedSubCommand(tmpDir.Root()), subCommand)
		})
	}
}
