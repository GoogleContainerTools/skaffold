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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildJibGradleToDocker(t *testing.T) {
	tests := []struct {
		description   string
		artifact      *latest.JibArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :jibDockerBuild --image=img:tag",
			),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibArtifact{Flags: []string{"--flag1", "--flag2"}},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :jibDockerBuild --image=img:tag --flag1 --flag2",
			),
		},
		{
			description: "build with project",
			artifact:    &latest.JibArtifact{Project: "project"},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :project:jibDockerBuild --image=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion="+MinimumJibGradleVersion+" :jibDockerBuild --image=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("build.gradle").Chdir()
			t.Override(&util.DefaultExecCommand, test.commands)
			api := (&testutil.FakeAPIClient{}).Add("img:tag", "imageID")
			localDocker := docker.NewLocalDaemon(api, nil, false, nil)

			builder := NewArtifactBuilder(localDocker, nil, false, false)
			result, err := builder.Build(context.Background(), ioutil.Discard, &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibArtifact: test.artifact,
				},
			}, "img:tag")

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("imageID", result)
			}
		})
	}
}

func TestBuildJibGradleToRegistry(t *testing.T) {
	tests := []struct {
		description   string
		artifact      *latest.JibArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "remote build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :jib --image=img:tag",
			),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibArtifact{Flags: []string{"--flag1", "--flag2"}},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :jib --image=img:tag --flag1 --flag2",
			),
		},
		{
			description: "build with project",
			artifact:    &latest.JibArtifact{Project: "project"},
			commands: testutil.CmdRun(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion=" + MinimumJibGradleVersion + " :project:jib --image=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"gradle -Djib.console=plain _skaffoldFailIfJibOutOfDate -Djib.requiredVersion="+MinimumJibGradleVersion+" :jib --image=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("build.gradle").Chdir()
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&docker.RemoteDigest, func(identifier string, _ map[string]bool) (string, error) {
				if identifier == "img:tag" {
					return "digest", nil
				}
				return "", errors.New("unknown remote tag")
			})
			localDocker := docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil)

			builder := NewArtifactBuilder(localDocker, nil, true, false)
			result, err := builder.Build(context.Background(), ioutil.Discard, &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					JibArtifact: test.artifact,
				},
			}, "img:tag")

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("digest", result)
			}
		})
	}
}

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
				strings.Join(getCommandGradle(ctx, tmpDir.Root(), &latest.JibArtifact{Project: "gradle-test"}).Args, " "),
				test.stdout,
				test.err,
			))

			// Change build file mod time
			os.Chtimes(build, test.modTime, test.modTime)

			deps, err := getDependenciesGradle(ctx, tmpDir.Root(), &latest.JibArtifact{Project: "gradle-test"})
			if test.err != nil {
				t.CheckErrorAndDeepEqual(true, err, "getting jib-gradle dependencies: initial Jib dependency refresh failed: failed to get Jib dependencies: "+test.err.Error(), err.Error())
			} else {
				t.CheckDeepEqual(test.expected, deps)
			}
		})
	}
}

func TestGetCommandGradle(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		description      string
		jibArtifact      latest.JibArtifact
		filesInWorkspace []string
		expectedCmd      func(workspace string) exec.Cmd
	}{
		{
			description:      "gradle default",
			jibArtifact:      latest.JibArtifact{},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:      "gradle default with project",
			jibArtifact:      latest.JibArtifact{Project: "project"},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:      "gradle with wrapper",
			jibArtifact:      latest.JibArtifact{},
			filesInWorkspace: []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q"})
			},
		},
		{
			description:      "gradle with wrapper and project",
			jibArtifact:      latest.JibArtifact{Project: "project"},
			filesInWorkspace: []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q"})
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Touch(test.filesInWorkspace...)

			cmd := getCommandGradle(ctx, tmpDir.Root(), &test.jibArtifact)

			expectedCmd := test.expectedCmd(tmpDir.Root())
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGenerateGradleArgs(t *testing.T) {
	tests := []struct {
		in                 latest.JibArtifact
		image              string
		skipTests          bool
		insecureRegistries map[string]bool
		out                []string
	}{
		{latest.JibArtifact{}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image"}},
		{latest.JibArtifact{Flags: []string{"-extra", "args"}}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image", "-extra", "args"}},
		{latest.JibArtifact{}, "image", true, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":task", "--image=image", "-x", "test"}},
		{latest.JibArtifact{Project: "project"}, "image", false, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "--image=image"}},
		{latest.JibArtifact{Project: "project"}, "image", true, nil, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "--image=image", "-x", "test"}},
		{latest.JibArtifact{Project: "project"}, "registry.tld/image", true, map[string]bool{"registry.tld": true}, []string{"-Djib.console=plain", "_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:task", "-Djib.allowInsecureRegistries=true", "--image=registry.tld/image", "-x", "test"}},
	}
	for _, test := range tests {
		command := GenerateGradleArgs("task", test.image, &test.in, test.skipTests, test.insecureRegistries)

		testutil.CheckDeepEqual(t, test.out, command)
	}
}
