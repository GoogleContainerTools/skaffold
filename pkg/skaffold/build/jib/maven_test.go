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

func TestBuildJibMavenToDocker(t *testing.T) {
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
				"mvn fake-mavenBuildArgs-for-dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "build with module",
			artifact:    &latest.JibArtifact{Project: "module"},
			commands: testutil.CmdRun(
				"mvn fake-mavenBuildArgs-for-module-for-dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"mvn fake-mavenBuildArgs-for-dockerBuild -Dimage=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenBuildArgsFunc, getMavenBuildArgsFuncFake(t, MinimumJibMavenVersion))
			t.NewTempDir().Touch("pom.xml").Chdir()
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

func TestBuildJibMavenToRegistry(t *testing.T) {
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
			commands:    testutil.CmdRun("mvn fake-mavenBuildArgs-for-build -Dimage=img:tag"),
		},
		{
			description: "build with module",
			artifact:    &latest.JibArtifact{Project: "module"},
			commands:    testutil.CmdRun("mvn fake-mavenBuildArgs-for-module-for-build -Dimage=img:tag"),
		},
		{
			description: "fail build",
			artifact:    &latest.JibArtifact{},
			commands: testutil.CmdRunErr(
				"mvn fake-mavenBuildArgs-for-build -Dimage=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenBuildArgsFunc, getMavenBuildArgsFuncFake(t, MinimumJibMavenVersion))
			t.NewTempDir().Touch("pom.xml").Chdir()
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

func TestMinimumMavenVersion(t *testing.T) {
	testutil.CheckDeepEqual(t, "1.4.0", MinimumJibMavenVersion)
}

func TestMavenWrapperDefinition(t *testing.T) {
	testutil.CheckDeepEqual(t, "mvn", MavenCommand.Executable)
	testutil.CheckDeepEqual(t, "mvnw", MavenCommand.Wrapper)
}

func TestGetDependenciesMaven(t *testing.T) {
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
			// Expected output differs from stdout since build file hasn't change, thus maven command won't run
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
				strings.Join(getCommandMaven(ctx, tmpDir.Root(), &latest.JibArtifact{Project: "maven-test"}).Args, " "),
				test.stdout,
				test.err,
			))

			// Change build file mod time
			if err := os.Chtimes(build, test.modTime, test.modTime); err != nil {
				t.Fatal(err)
			}

			deps, err := getDependenciesMaven(ctx, tmpDir.Root(), &latest.JibArtifact{Project: "maven-test"})
			if test.err != nil {
				t.CheckErrorAndDeepEqual(true, err, "getting jib-maven dependencies: initial Jib dependency refresh failed: failed to get Jib dependencies: "+test.err.Error(), err.Error())
			} else {
				t.CheckDeepEqual(test.expected, deps)
			}
		})
	}
}

func TestGetCommandMaven(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		description      string
		jibArtifact      latest.JibArtifact
		filesInWorkspace []string
		expectedCmd      func(workspace string) exec.Cmd
	}{
		{
			description:      "maven basic",
			jibArtifact:      latest.JibArtifact{},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenArgs", "jib:_skaffold-files-v2", "--quiet", "--batch-mode"})
			},
		},
		{
			description:      "maven with wrapper",
			jibArtifact:      latest.JibArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenArgs", "jib:_skaffold-files-v2", "--quiet", "--batch-mode"})
			},
		},
		{
			description:      "maven with multi-modules",
			jibArtifact:      latest.JibArtifact{Project: "module"},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenArgs-for-module", "jib:_skaffold-files-v2", "--quiet", "--batch-mode"})
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenArgsFunc, getMavenArgsFuncFake(t, MinimumJibMavenVersion))
			tmpDir := t.NewTempDir().
				Touch(test.filesInWorkspace...)

			cmd := getCommandMaven(ctx, tmpDir.Root(), &test.jibArtifact)

			expectedCmd := test.expectedCmd(tmpDir.Root())
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGetSyncMapCommandMaven(t *testing.T) {
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
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenBuildArgs-for-_skaffold-sync-map-skipTests"})
			},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "module"},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenBuildArgs-for-module-for-_skaffold-sync-map-skipTests"})
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenBuildArgsFunc, getMavenBuildArgsFuncFake(t, MinimumJibMavenVersionForSync))
			cmd := getSyncMapCommandMaven(ctx, test.workspace, &test.jibArtifact)
			expectedCmd := test.expectedCmd(test.workspace)
			t.CheckDeepEqual(expectedCmd.Path, cmd.Path)
			t.CheckDeepEqual(expectedCmd.Args, cmd.Args)
			t.CheckDeepEqual(expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGenerateMavenBuildArgs(t *testing.T) {
	tests := []struct {
		description        string
		a                  latest.JibArtifact
		image              string
		skipTests          bool
		insecureRegistries map[string]bool
		out                []string
	}{
		{"single module", latest.JibArtifact{}, "image", false, nil, []string{"fake-mavenBuildArgs-for-test-goal", "-Dimage=image"}},
		{"single module without tests", latest.JibArtifact{}, "image", true, nil, []string{"fake-mavenBuildArgs-for-test-goal-skipTests", "-Dimage=image"}},
		{"multi module", latest.JibArtifact{Project: "module"}, "image", false, nil, []string{"fake-mavenBuildArgs-for-module-for-test-goal", "-Dimage=image"}},
		{"multi module without tests", latest.JibArtifact{Project: "module"}, "image", true, nil, []string{"fake-mavenBuildArgs-for-module-for-test-goal-skipTests", "-Dimage=image"}},
		{"multi module without tests with insecure-registry", latest.JibArtifact{Project: "module"}, "registry.tld/image", true, map[string]bool{"registry.tld": true}, []string{"fake-mavenBuildArgs-for-module-for-test-goal-skipTests", "-Djib.allowInsecureRegistries=true", "-Dimage=registry.tld/image"}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenBuildArgsFunc, getMavenBuildArgsFuncFake(t, MinimumJibMavenVersion))
			args := GenerateMavenBuildArgs("test-goal", test.image, &test.a, test.skipTests, test.insecureRegistries)
			t.CheckDeepEqual(test.out, args)
		})
	}
}

func TestMavenBuildArgs(t *testing.T) {
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
			expected:    []string{"-Djib.console=plain", "fake-mavenArgs", "prepare-package", "jib:test-goal"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latest.JibArtifact{},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-mavenArgs", "-DskipTests=true", "prepare-package", "jib:test-goal"},
		},
		{
			description: "single module plain console",
			jibArtifact: latest.JibArtifact{},
			skipTests:   true,
			showColors:  false,
			expected:    []string{"--batch-mode", "fake-mavenArgs", "-DskipTests=true", "prepare-package", "jib:test-goal"},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "module"},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-mavenArgs-for-module", "package", "jib:test-goal", "-Djib.containerize=module"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latest.JibArtifact{Project: "module"},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Djib.console=plain", "fake-mavenArgs-for-module", "-DskipTests=true", "package", "jib:test-goal", "-Djib.containerize=module"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenArgsFunc, getMavenArgsFuncFake(t, "test-version"))
			args := mavenBuildArgs("test-goal", &test.jibArtifact, test.skipTests, test.showColors, "test-version")
			t.CheckDeepEqual(test.expected, args)
		})
	}
}

func TestMavenArgs(t *testing.T) {
	tests := []struct {
		description string
		jibArtifact latest.JibArtifact
		expected    []string
	}{
		{
			description: "single module",
			jibArtifact: latest.JibArtifact{},
			expected:    []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--non-recursive"},
		},
		{
			description: "single module with extra flags",
			jibArtifact: latest.JibArtifact{
				Flags: []string{"--flag1", "--flag2"},
			},
			expected: []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--flag1", "--flag2", "--non-recursive"},
		},
		{
			description: "multi module",
			jibArtifact: latest.JibArtifact{Project: "module"},
			expected:    []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--projects", "module", "--also-make"},
		},
		{
			description: "multi module with extra falgs",
			jibArtifact: latest.JibArtifact{
				Project: "module",
				Flags:   []string{"--flag1", "--flag2"},
			},
			expected: []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--flag1", "--flag2", "--projects", "module", "--also-make"},
		},
	}
	for _, test := range tests {
		args := mavenArgs(&test.jibArtifact, "test-version")
		testutil.CheckDeepEqual(t, test.expected, args)
	}
}

func getMavenArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(*latest.JibArtifact, string) []string {
	return func(a *latest.JibArtifact, minimumVersion string) []string {
		t.CheckDeepEqual(expectedMinimumVersion, minimumVersion)
		if a.Project == "" {
			return []string{"fake-mavenArgs"}
		}
		return []string{"fake-mavenArgs-for-" + a.Project}
	}
}

func getMavenBuildArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(string, *latest.JibArtifact, bool, bool, string) []string {
	return func(goal string, a *latest.JibArtifact, skipTests, showColors bool, minimumVersion string) []string {
		t.CheckDeepEqual(expectedMinimumVersion, minimumVersion)
		testString := ""
		if skipTests {
			testString = "-skipTests"
		}

		if a.Project == "" {
			return []string{"fake-mavenBuildArgs-for-" + goal + testString}
		}
		return []string{"fake-mavenBuildArgs-for-" + a.Project + "-for-" + goal + testString}
	}
}
