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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestMinimumGradleVersion(t *testing.T) {
	testutil.CheckDeepEqual(t, "1.4.0", MinimumJibGradleVersion)
}

func TestGradleWrapperDefinition(t *testing.T) {
	testutil.CheckDeepEqual(t, "gradle", GradleCommand.Executable)
	testutil.CheckDeepEqual(t, "gradlew", GradleCommand.Wrapper)
}

func TestGetDependenciesGradle(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Touch("build", "dep1", "dep2")
	build := tmpDir.Path("build")
	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dep2")

	ctx := context.Background()

	tests := []struct {
		description string
		stdout      string
		modTime     time.Time
		expected    []string
		err         error
	}{
		{
			description: "failure",
			stdout:      "",
			modTime:     time.Unix(0, 0),
			err:         errors.New("error"),
		},
		{
			description: "success",
			stdout:      fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[]}", build, dep1),
			modTime:     time.Unix(0, 0),
			expected:    []string{"build", "dep1"},
		},
		{
			// Expected output differs from stdout since build file hasn't change, thus gradle command won't run
			description: "success",
			stdout:      fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\", \"%s\"],\"ignore\":[]}", build, dep1, dep2),
			modTime:     time.Unix(0, 0),
			expected:    []string{"build", "dep1"},
		},
		{
			description: "success",
			stdout:      fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\", \"%s\"],\"ignore\":[]}", build, dep1, dep2),
			modTime:     time.Unix(10000, 0),
			expected:    []string{"build", "dep1", "dep2"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOutErr(
				strings.Join(getCommandGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{Project: "gradle-test"}).Args, " "),
				test.stdout,
				test.err,
			))

			// Change build file mod time
			os.Chtimes(build, test.modTime, test.modTime)

			deps, err := GetDependenciesGradle(ctx, tmpDir.Root(), &latest.JibGradleArtifact{Project: "gradle-test"})
			if test.err != nil {
				t.CheckErrorAndDeepEqual(true, err, "getting jibGradle dependencies: initial Jib dependency refresh failed: failed to get Jib dependencies: "+test.err.Error(), err.Error())
			} else {
				t.CheckDeepEqual(test.expected, deps)
			}
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		description       string
		jibGradleArtifact latest.JibGradleArtifact
		filesInWorkspace  []string
		expectedCmd       func(workspace string) exec.Cmd
	}{
		{
			description:       "gradle default",
			jibGradleArtifact: latest.JibGradleArtifact{},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:       "gradle default with project",
			jibGradleArtifact: latest.JibGradleArtifact{Project: "project"},
			filesInWorkspace:  []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:       "gradle with wrapper",
			jibGradleArtifact: latest.JibGradleArtifact{},
			filesInWorkspace:  []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:       "gradle with wrapper and project",
			jibGradleArtifact: latest.JibGradleArtifact{Project: "project"},
			filesInWorkspace:  []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q"})
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Touch(test.filesInWorkspace...)

			cmd := getCommandGradle(ctx, tmpDir.Root(), &test.jibGradleArtifact)

			expectedCmd := test.expectedCmd(tmpDir.Root())
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGenerateGradleArgs(t *testing.T) {
	tests := []struct {
		in                 latest.JibGradleArtifact
		image              string
		skipTests          bool
		insecureRegistries map[string]bool
		out                []string
	}{
		{latest.JibGradleArtifact{}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image"}},
		{latest.JibGradleArtifact{Flags: []string{"-extra", "args"}}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image", "-extra", "args"}},
		{latest.JibGradleArtifact{}, "image", true, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image", "-x", "test"}},
		{latest.JibGradleArtifact{Project: "project"}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "--image=image"}},
		{latest.JibGradleArtifact{Project: "project"}, "image", true, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "--image=image", "-x", "test"}},
		{latest.JibGradleArtifact{Project: "project"}, "registry.tld/image", true, map[string]bool{"registry.tld": true}, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "-Djib.allowInsecureRegistries=true", "--image=registry.tld/image", "-x", "test"}},
	}
	for _, test := range tests {
		command := GenerateGradleArgs("task", test.image, &test.in, test.skipTests, test.insecureRegistries)

		testutil.CheckDeepEqual(t, test.out, command)
	}
}
