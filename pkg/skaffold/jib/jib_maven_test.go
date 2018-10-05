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
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestMavenWrapperDefinition(t *testing.T) {
	if MavenCommand.Executable != "mvn" {
		t.Error("GradleCommand executable should be 'mvn'")
	}
	if MavenCommand.Wrapper != "mvnw" {
		t.Error("MavenCommand wrapper should be 'mvnw'")
	}
}

func TestGetDependenciesMaven(t *testing.T) {
	var tests = []struct {
		description string
		stdout      string
		err         error
	}{
		{
			description: "success",
			stdout:      "dep1\ndep2\n\n\n",
			err:         nil,
		},
		{
			description: "failure",
			stdout:      "",
			err:         errors.New("error"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmdOut(
				strings.Join(getCommandMaven(tmpDir.Root(), &latest.JibMavenArtifact{}).Args, " "),
				test.stdout,
				test.err,
			)

			deps, err := GetDependenciesMaven(tmpDir.Root(), &latest.JibMavenArtifact{})
			if test.err != nil {
				testutil.CheckErrorAndDeepEqual(t, true, err, "getting jib-maven dependencies: "+test.err.Error(), err.Error())
			} else {
				testutil.CheckDeepEqual(t, []string{"dep1", "dep2"}, deps)
			}
		})
	}
}

func TestGetCommandMaven(t *testing.T) {
	var tests = []struct {
		description      string
		jibMavenArtifact latest.JibMavenArtifact
		filesInWorkspace []string
		expectedCmd      func(workspace string) *exec.Cmd
	}{
		{
			description:      "maven no profile",
			jibMavenArtifact: latest.JibMavenArtifact{},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(workspace, []string{"jib:_skaffold-files", "-q"})
			},
		},
		{
			description:      "maven with profile",
			jibMavenArtifact: latest.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(workspace, []string{"jib:_skaffold-files", "-q", "-P", "profile"})
			},
		},
		{
			description:      "maven with wrapper no profile",
			jibMavenArtifact: latest.JibMavenArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(workspace, []string{"jib:_skaffold-files", "-q"})
			},
		},
		{
			description:      "maven with wrapper no profile",
			jibMavenArtifact: latest.JibMavenArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.cmd"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(workspace, []string{"jib:_skaffold-files", "-q"})
			},
		},
		{
			description:      "maven with wrapper and profile",
			jibMavenArtifact: latest.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(workspace, []string{"jib:_skaffold-files", "-q", "-P", "profile"})
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

			cmd := getCommandMaven(tmpDir.Root(), &test.jibMavenArtifact)
			expectedCmd := test.expectedCmd(tmpDir.Root())
			testutil.CheckDeepEqual(t, expectedCmd.Path, cmd.Path)
			testutil.CheckDeepEqual(t, expectedCmd.Args, cmd.Args)
			testutil.CheckDeepEqual(t, expectedCmd.Dir, cmd.Dir)
		})
	}
}
