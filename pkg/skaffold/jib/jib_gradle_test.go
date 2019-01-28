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
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestGradleWrapperDefinition(t *testing.T) {
	testutil.CheckDeepEqual(t, "gradle", GradleCommand.Executable)
	testutil.CheckDeepEqual(t, "gradlew", GradleCommand.Wrapper)
}

func TestGetDependenciesGradle(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("dep1", "")
	tmpDir.Write("dep2", "")

	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dep2")

	ctx := context.Background()

	var tests = []struct {
		description string
		stdout      string
		expected    []string
		err         error
	}{
		{
			description: "success",
			stdout:      fmt.Sprintf("%s\n%s\n\n\n", dep1, dep2),
			expected:    []string{dep1, dep2},
		},
		{
			description: "failure",
			stdout:      "",
			err:         errors.New("error"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOutErr(
				strings.Join(getCommandGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{}).Args, " "),
				test.stdout,
				test.err,
			)

			deps, err := GetDependenciesGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{})
			if test.err != nil {
				testutil.CheckErrorAndDeepEqual(t, true, err, "getting jibGradle dependencies: "+test.err.Error(), err.Error())
			} else {
				testutil.CheckDeepEqual(t, []string{dep1, dep2}, deps)
			}
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	ctx := context.Background()

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
				return GradleCommand.CreateCommand(ctx, workspace, []string{":_jibSkaffoldFiles", "-q"})
			},
		},
		{
			description:       "gradle default with project",
			jibGradleArtifact: latest.JibGradleArtifact{Project: "project"},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{":project:_jibSkaffoldFiles", "-q"})
			},
		},
		{
			description:       "gradle with wrapper",
			jibGradleArtifact: latest.JibGradleArtifact{},
			filesInWorkspace:  []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{":_jibSkaffoldFiles", "-q"})
			},
		},
		{
			description:       "gradle with wrapper and project",
			jibGradleArtifact: latest.JibGradleArtifact{Project: "project"},
			filesInWorkspace:  []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{":project:_jibSkaffoldFiles", "-q"})
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

func TestGenerateGradleArgs(t *testing.T) {
	var testCases = []struct {
		in  latest.JibGradleArtifact
		out []string
	}{
		{latest.JibGradleArtifact{}, []string{":task", "--image=image"}},
		{latest.JibGradleArtifact{Project: "project"}, []string{":project:task", "--image=image"}},
	}

	for _, tt := range testCases {
		command := GenerateGradleArgs("task", "image", &tt.in)

		testutil.CheckDeepEqual(t, tt.out, command)
	}
}
