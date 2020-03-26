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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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
				"gradle fake-gradleBuildArgs-for-jibDockerBuild --image=img:tag",
			),
		},
		{
			description: "build with project",
			artifact:    &latest.JibArtifact{Project: "project"},
			commands: testutil.CmdRun(
				"gradle fake-gradleBuildArgs-for-project-for-jibDockerBuild --image=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"gradle fake-gradleBuildArgs-for-jibDockerBuild --image=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("build.gradle").Chdir()
			t.Override(&gradleBuildArgsFunc, getGradleBuildArgsFuncFake(t, MinimumJibGradleVersion))
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
				"gradle fake-gradleBuildArgs-for-jib --image=img:tag",
			),
		},
		{
			description: "build with project",
			artifact:    &latest.JibArtifact{Project: "project"},
			commands: testutil.CmdRun(
				"gradle fake-gradleBuildArgs-for-project-for-jib --image=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"gradle fake-gradleBuildArgs-for-jib --image=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("build.gradle").Chdir()
			t.Override(&gradleBuildArgsFunc, getGradleBuildArgsFuncFake(t, MinimumJibGradleVersion))
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
	tmpDir := testutil.NewTempDir(t)

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
			if err := os.Chtimes(build, test.modTime, test.modTime); err != nil {
				t.Fatal(err)
			}

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
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q", "--console=plain"})
			},
		},
		{
			description:      "gradle default with project",
			jibArtifact:      latest.JibArtifact{Project: "project"},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q", "--console=plain"})
			},
		},
		{
			description:      "gradle with wrapper",
			jibArtifact:      latest.JibArtifact{},
			filesInWorkspace: []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":_jibSkaffoldFilesV2", "-q", "--console=plain"})
			},
		},
		{
			description:      "gradle with wrapper and project",
			jibArtifact:      latest.JibArtifact{Project: "project"},
			filesInWorkspace: []string{"gradlew", "gradlew.cmd"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=" + MinimumJibGradleVersion, ":project:_jibSkaffoldFilesV2", "-q", "--console=plain"})
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

func TestGetSyncMapCommandGradle(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		description string
		workspace   string
		jibArtifact latest.JibArtifact
		expectedCmd func(workspace string) exec.Cmd
	}{
		{
			description: "single module",
			jibArtifact: latest.JibArtifact{},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"fake-gradleBuildArgs-for-_jibSkaffoldSyncMap-skipTests"})
			},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "project"},
			expectedCmd: func(workspace string) exec.Cmd {
				return GradleCommand.CreateCommand(ctx, workspace, []string{"fake-gradleBuildArgs-for-project-for-_jibSkaffoldSyncMap-skipTests"})
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&gradleBuildArgsFunc, getGradleBuildArgsFuncFake(t, MinimumJibGradleVersionForSync))
			cmd := getSyncMapCommandGradle(ctx, test.workspace, &test.jibArtifact)
			expectedCmd := test.expectedCmd(test.workspace)
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGenerateGradleBuildArgs(t *testing.T) {
	tests := []struct {
		description        string
		in                 latest.JibArtifact
		image              string
		skipTests          bool
		insecureRegistries map[string]bool
		out                []string
	}{
		{"single module", latest.JibArtifact{}, "image", false, nil, []string{"fake-gradleBuildArgs-for-testTask", "--image=image"}},
		{"single module without tests", latest.JibArtifact{}, "image", true, nil, []string{"fake-gradleBuildArgs-for-testTask-skipTests", "--image=image"}},
		{"multi module", latest.JibArtifact{Project: "project"}, "image", false, nil, []string{"fake-gradleBuildArgs-for-project-for-testTask", "--image=image"}},
		{"multi module without tests", latest.JibArtifact{Project: "project"}, "image", true, nil, []string{"fake-gradleBuildArgs-for-project-for-testTask-skipTests", "--image=image"}},
		{"multi module without tests with insecure registries", latest.JibArtifact{Project: "project"}, "registry.tld/image", true, map[string]bool{"registry.tld": true}, []string{"fake-gradleBuildArgs-for-project-for-testTask-skipTests", "-Djib.allowInsecureRegistries=true", "--image=registry.tld/image"}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&gradleBuildArgsFunc, getGradleBuildArgsFuncFake(t, MinimumJibGradleVersion))
			command := GenerateGradleBuildArgs("testTask", test.image, &test.in, test.skipTests, test.insecureRegistries)
			t.CheckDeepEqual(test.out, command)
		})
	}
}

func TestGradleArgs(t *testing.T) {
	tests := []struct {
		description string
		jibArtifact latest.JibArtifact
		expected    []string
	}{
		{
			description: "single module",
			jibArtifact: latest.JibArtifact{},
			expected:    []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=test-version", ":testTask"},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "module"},
			expected:    []string{"_skaffoldFailIfJibOutOfDate", "-Djib.requiredVersion=test-version", ":module:testTask"},
		},
	}
	for _, test := range tests {
		args := gradleArgs(&test.jibArtifact, "testTask", "test-version")
		testutil.CheckDeepEqual(t, test.expected, args)
	}
}

func TestGradleBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		jibArtifact latest.JibArtifact
		skipTests   bool
		showColors  bool
		expected    []string
	}{
		{
			description: "single module",
			jibArtifact: latest.JibArtifact{},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-testTask"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latest.JibArtifact{},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-testTask", "-x", "test"},
		},
		{
			description: "single module plain console",
			jibArtifact: latest.JibArtifact{},
			skipTests:   true,
			showColors:  false,
			expected:    []string{"--console=plain", "fake-gradleArgs-for-testTask", "-x", "test"},
		},
		{
			description: "single module with extra flags",
			jibArtifact: latest.JibArtifact{Flags: []string{"--flag1", "--flag2"}},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-testTask", "--flag1", "--flag2"},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "module"},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-module-for-testTask"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latest.JibArtifact{Project: "module"},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-module-for-testTask", "-x", "test"},
		},
		{
			description: "multi module with extra flags",
			jibArtifact: latest.JibArtifact{Project: "module", Flags: []string{"--flag1", "--flag2"}},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-gradleArgs-for-module-for-testTask", "--flag1", "--flag2"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&gradleArgsFunc, getGradleArgsFuncFake(t, "test-version"))
			args := gradleBuildArgs("testTask", &test.jibArtifact, test.skipTests, test.showColors, "test-version")
			t.CheckDeepEqual(test.expected, args)
		})
	}
}

func getGradleArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(*latest.JibArtifact, string, string) []string {
	return func(a *latest.JibArtifact, task string, minimumVersion string) []string {
		t.CheckDeepEqual(expectedMinimumVersion, minimumVersion)
		if a.Project == "" {
			return []string{"fake-gradleArgs-for-" + task}
		}
		return []string{"fake-gradleArgs-for-" + a.Project + "-for-" + task}
	}
}

// check that parameters are actually passed though
func getGradleBuildArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(string, *latest.JibArtifact, bool, bool, string) []string {
	return func(task string, a *latest.JibArtifact, skipTests, showColors bool, minimumVersion string) []string {
		t.CheckDeepEqual(expectedMinimumVersion, minimumVersion)

		testString := ""
		if skipTests {
			testString = "-skipTests"
		}

		if a.Project == "" {
			return []string{"fake-gradleBuildArgs-for-" + task + testString}
		}
		return []string{"fake-gradleBuildArgs-for-" + a.Project + "-for-" + task + testString}
	}
}
