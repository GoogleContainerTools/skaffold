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
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestGradleWrapperDefinition(t *testing.T) {
	if GradleCommand.Executable != "gradle" {
		t.Error("GradleCommand executable should be 'gradle'")
	}
	if GradleCommand.Wrapper != "gradlew" {
		t.Error("GradleCommand wrapper should be 'gradlew'")
	}
}

func TestGetDependenciesGradle(t *testing.T) {
	ctx := context.TODO()

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
				strings.Join(getCommandGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{}).Args, " "),
				test.stdout,
				test.err,
			)

			deps, err := GetDependenciesGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{})
			if test.err != nil {
				testutil.CheckErrorAndDeepEqual(t, true, err, "getting jib-gradle dependencies: "+test.err.Error(), err.Error())
			} else {
				testutil.CheckDeepEqual(t, []string{"dep1", "dep2"}, deps)
			}
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	ctx := context.TODO()

	var tests = []struct {
		description       string
		jibGradleArtifact latest.JibGradleArtifact
		filesInWorkspace  []string
		expectedCmd       func(workspace string) *exec.Cmd
	}{
		{
			description:       "gradle default",
			jibGradleArtifact: latest.JibGradleArtifact{},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_jibSkaffoldFiles", "-q"})
			},
		},
		{
			description:       "gradle with wrapper",
			jibGradleArtifact: latest.JibGradleArtifact{},
			filesInWorkspace:  []string{"gradle", "gradle.cmd"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_jibSkaffoldFiles", "-q"})
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

			cmd := getCommandGradle(ctx, tmpDir.Root(), &test.jibGradleArtifact)
			expectedCmd := test.expectedCmd(tmpDir.Root())
			testutil.CheckDeepEqual(t, expectedCmd.Path, cmd.Path)
			testutil.CheckDeepEqual(t, expectedCmd.Args, cmd.Args)
			testutil.CheckDeepEqual(t, expectedCmd.Dir, cmd.Dir)
		})
	}
}
