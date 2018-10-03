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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestGetDependenciesGradle(t *testing.T) {
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
				strings.Join(getCommandGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}).Args, " "),
				test.stdout,
				test.err,
			)

			deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{})
			if test.err != nil {
				testutil.CheckErrorAndDeepEqual(t, true, err, "getting jib-gradle dependencies: "+test.err.Error(), err.Error())
			} else {
				testutil.CheckDeepEqual(t, []string{"dep1", "dep2"}, deps)
			}
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	var tests = []struct {
		description       string
		jibGradleArtifact v1alpha3.JibGradleArtifact
		filesInWorkspace  []string
		expectedCmd       func(workspace string) *exec.Cmd
	}{
		{
			description:       "gradle default",
			jibGradleArtifact: v1alpha3.JibGradleArtifact{},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return getCommand(workspace, "gradle", "ignored", []string{"_jibSkaffoldFiles", "-q"})
			},
		},
		{
			description:       "gradle with wrapper",
			jibGradleArtifact: v1alpha3.JibGradleArtifact{},
			filesInWorkspace:  []string{getWrapperGradle()},
			expectedCmd: func(workspace string) *exec.Cmd {
				return getCommand(workspace, "ignored", getWrapperGradle(), []string{"_jibSkaffoldFiles", "-q"})
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

			cmd := getCommandGradle(tmpDir.Root(), &test.jibGradleArtifact)
			expectedCmd := test.expectedCmd(tmpDir.Root())
			testutil.CheckDeepEqual(t, expectedCmd.Path, cmd.Path)
			testutil.CheckDeepEqual(t, expectedCmd.Args, cmd.Args)
			testutil.CheckDeepEqual(t, expectedCmd.Dir, cmd.Dir)
		})
	}
}
