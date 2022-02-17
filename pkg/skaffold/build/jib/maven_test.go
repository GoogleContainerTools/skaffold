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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildJibMavenToDocker(t *testing.T) {
	tests := []struct {
		description   string
		artifact      *latestV2.JibArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latestV2.JibArtifact{},
			commands: testutil.CmdRun(
				"mvn fake-mavenBuildArgs-for-dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "build with module",
			artifact:    &latestV2.JibArtifact{Project: "module"},
			commands: testutil.CmdRun(
				"mvn fake-mavenBuildArgs-for-module-for-dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "build with custom base image",
			artifact:    &latestV2.JibArtifact{BaseImage: "docker://busybox"},
			commands: testutil.CmdRun(
				"mvn fake-mavenBuildArgs-for-dockerBuild -Djib.from.image=docker://busybox -Dimage=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latestV2.JibArtifact{},
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
			localDocker := fakeLocalDaemon(api)

			builder := NewArtifactBuilder(localDocker, &mockConfig{}, false, false, mockArtifactResolver{})
			result, err := builder.Build(context.Background(), ioutil.Discard, &latestV2.Artifact{
				ArtifactType: latestV2.ArtifactType{
					JibArtifact: test.artifact,
				},
			}, "img:tag", platform.All)

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
		artifact      *latestV2.JibArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latestV2.JibArtifact{},
			commands:    testutil.CmdRun("mvn fake-mavenBuildArgs-for-build -Dimage=img:tag"),
		},
		{
			description: "build with module",
			artifact:    &latestV2.JibArtifact{Project: "module"},
			commands:    testutil.CmdRun("mvn fake-mavenBuildArgs-for-module-for-build -Dimage=img:tag"),
		},
		{
			description: "build with custom base image",
			artifact:    &latestV2.JibArtifact{BaseImage: "docker://busybox"},
			commands:    testutil.CmdRun("mvn fake-mavenBuildArgs-for-build -Djib.from.image=docker://busybox -Dimage=img:tag"),
		},
		{
			description: "fail build",
			artifact:    &latestV2.JibArtifact{},
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
			t.Override(&docker.RemoteDigest, func(identifier string, _ docker.Config) (string, error) {
				if identifier == "img:tag" {
					return "digest", nil
				}
				return "", errors.New("unknown remote tag")
			})
			localDocker := fakeLocalDaemon(&testutil.FakeAPIClient{})

			builder := NewArtifactBuilder(localDocker, &mockConfig{}, true, false, mockArtifactResolver{})
			result, err := builder.Build(context.Background(), ioutil.Discard, &latestV2.Artifact{
				ArtifactType: latestV2.ArtifactType{
					JibArtifact: test.artifact,
				},
			}, "img:tag", platform.All)

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
				strings.Join(getCommandMaven(ctx, tmpDir.Root(), &latestV2.JibArtifact{Project: "maven-test"}).Args, " "),
				test.stdout,
				test.err,
			))

			// Change build file mod time
			if err := os.Chtimes(build, test.modTime, test.modTime); err != nil {
				t.Fatal(err)
			}
			ws := tmpDir.Root()
			deps, err := getDependenciesMaven(ctx, ws, &latestV2.JibArtifact{Project: "maven-test"})
			if test.err != nil {
				prefix := fmt.Sprintf("could not fetch dependencies for workspace %s: initial Jib dependency refresh failed: failed to get Jib dependencies: ", ws)
				t.CheckErrorAndDeepEqual(true, err, prefix+test.err.Error(), err.Error())
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
		jibArtifact      latestV2.JibArtifact
		filesInWorkspace []string
		expectedCmd      func(workspace string) exec.Cmd
	}{
		{
			description:      "maven basic",
			jibArtifact:      latestV2.JibArtifact{},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenArgs", "jib:_skaffold-files-v2", "--quiet", "--batch-mode"})
			},
		},
		{
			description:      "maven with wrapper",
			jibArtifact:      latestV2.JibArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenArgs", "jib:_skaffold-files-v2", "--quiet", "--batch-mode"})
			},
		},
		{
			description:      "maven with multi-modules",
			jibArtifact:      latestV2.JibArtifact{Project: "module"},
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
		jibArtifact latestV2.JibArtifact
		expectedCmd func(workspace string) exec.Cmd
	}{
		{
			description: "single module",
			jibArtifact: latestV2.JibArtifact{},
			expectedCmd: func(workspace string) exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"fake-mavenBuildArgs-for-_skaffold-sync-map-skipTests"})
			},
		},
		{
			description: "multi module",
			jibArtifact: latestV2.JibArtifact{Project: "module"},
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
		a                  latestV2.JibArtifact
		deps               []*latestV2.ArtifactDependency
		image              string
		r                  ArtifactResolver
		skipTests          bool
		pushImages         bool
		insecureRegistries map[string]bool
		out                []string
	}{
		{description: "single module", image: "image", out: []string{"fake-mavenBuildArgs-for-test-goal", "-Dimage=image"}},
		{description: "single module without tests", image: "image", skipTests: true, out: []string{"fake-mavenBuildArgs-for-test-goal-skipTests", "-Dimage=image"}},
		{description: "multi module", a: latestV2.JibArtifact{Project: "module"}, image: "image", out: []string{"fake-mavenBuildArgs-for-module-for-test-goal", "-Dimage=image"}},
		{description: "multi module without tests", a: latestV2.JibArtifact{Project: "module"}, image: "image", skipTests: true, out: []string{"fake-mavenBuildArgs-for-module-for-test-goal-skipTests", "-Dimage=image"}},
		{description: "multi module without tests with insecure-registry", a: latestV2.JibArtifact{Project: "module"}, image: "registry.tld/image", skipTests: true, insecureRegistries: map[string]bool{"registry.tld": true}, out: []string{"fake-mavenBuildArgs-for-module-for-test-goal-skipTests", "-Djib.allowInsecureRegistries=true", "-Dimage=registry.tld/image"}},
		{description: "single module with custom base image", a: latestV2.JibArtifact{BaseImage: "docker://busybox"}, image: "image", out: []string{"fake-mavenBuildArgs-for-test-goal", "-Djib.from.image=docker://busybox", "-Dimage=image"}},
		{description: "multi module with custom base image", a: latestV2.JibArtifact{Project: "module", BaseImage: "docker://busybox"}, image: "image", out: []string{"fake-mavenBuildArgs-for-module-for-test-goal", "-Djib.from.image=docker://busybox", "-Dimage=image"}},
		{
			description: "single module with local base image from required artifacts",
			a:           latestV2.JibArtifact{BaseImage: "alias"},
			deps:        []*latestV2.ArtifactDependency{{ImageName: "img", Alias: "alias"}},
			image:       "image",
			r:           mockArtifactResolver{m: map[string]string{"img": "img:tag"}},
			out:         []string{"fake-mavenBuildArgs-for-test-goal", "-Djib.from.image=docker://img:tag", "-Dimage=image"},
		},
		{
			description: "multi module with local base image from required artifacts",
			a:           latestV2.JibArtifact{Project: "module", BaseImage: "alias"},
			deps:        []*latestV2.ArtifactDependency{{ImageName: "img", Alias: "alias"}},
			image:       "image",
			r:           mockArtifactResolver{m: map[string]string{"img": "img:tag"}},
			out:         []string{"fake-mavenBuildArgs-for-module-for-test-goal", "-Djib.from.image=docker://img:tag", "-Dimage=image"},
		},
		{
			description: "single module with remote base image from required artifacts",
			a:           latestV2.JibArtifact{BaseImage: "alias"},
			deps:        []*latestV2.ArtifactDependency{{ImageName: "img", Alias: "alias"}},
			image:       "image",
			pushImages:  true,
			r:           mockArtifactResolver{m: map[string]string{"img": "img:tag"}},
			out:         []string{"fake-mavenBuildArgs-for-test-goal", "-Djib.from.image=img:tag", "-Dimage=image"},
		},
		{
			description: "multi module with remote base image from required artifacts",
			a:           latestV2.JibArtifact{Project: "module", BaseImage: "alias"},
			deps:        []*latestV2.ArtifactDependency{{ImageName: "img", Alias: "alias"}},
			image:       "image",
			pushImages:  true,
			r:           mockArtifactResolver{m: map[string]string{"img": "img:tag"}},
			out:         []string{"fake-mavenBuildArgs-for-module-for-test-goal", "-Djib.from.image=img:tag", "-Dimage=image"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&mavenBuildArgsFunc, getMavenBuildArgsFuncFake(t, MinimumJibMavenVersion))
			args := GenerateMavenBuildArgs("test-goal", test.image, &test.a, test.skipTests, test.pushImages, test.deps, test.r, test.insecureRegistries, false)
			t.CheckDeepEqual(test.out, args)
		})
	}
}

func TestMavenBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		jibArtifact latestV2.JibArtifact
		skipTests   bool
		showColors  bool
		expected    []string
	}{
		{
			description: "single module",
			jibArtifact: latestV2.JibArtifact{},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Dstyle.color=always", "-Djansi.passthrough=true", "-Djib.console=plain", "fake-mavenArgs", "prepare-package", "jib:test-goal"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latestV2.JibArtifact{},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Dstyle.color=always", "-Djansi.passthrough=true", "-Djib.console=plain", "fake-mavenArgs", "-DskipTests=true", "prepare-package", "jib:test-goal"},
		},
		{
			description: "single module plain console",
			jibArtifact: latestV2.JibArtifact{},
			skipTests:   true,
			showColors:  false,
			expected:    []string{"--batch-mode", "fake-mavenArgs", "-DskipTests=true", "prepare-package", "jib:test-goal"},
		},
		{
			description: "multi module",
			jibArtifact: latestV2.JibArtifact{Project: "module"},
			skipTests:   false,
			showColors:  true,
			expected:    []string{"-Dstyle.color=always", "-Djansi.passthrough=true", "-Djib.console=plain", "fake-mavenArgs-for-module", "package", "jib:test-goal", "-Djib.containerize=module"},
		},
		{
			description: "single module skip tests",
			jibArtifact: latestV2.JibArtifact{Project: "module"},
			skipTests:   true,
			showColors:  true,
			expected:    []string{"-Dstyle.color=always", "-Djansi.passthrough=true", "-Djib.console=plain", "fake-mavenArgs-for-module", "-DskipTests=true", "package", "jib:test-goal", "-Djib.containerize=module"},
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
		jibArtifact latestV2.JibArtifact
		expected    []string
	}{
		{
			description: "single module",
			jibArtifact: latestV2.JibArtifact{},
			expected:    []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--non-recursive"},
		},
		{
			description: "single module with extra flags",
			jibArtifact: latestV2.JibArtifact{
				Flags: []string{"--flag1", "--flag2"},
			},
			expected: []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--flag1", "--flag2", "--non-recursive"},
		},
		{
			description: "multi module",
			jibArtifact: latestV2.JibArtifact{Project: "module"},
			expected:    []string{"jib:_skaffold-fail-if-jib-out-of-date", "-Djib.requiredVersion=test-version", "--projects", "module", "--also-make"},
		},
		{
			description: "multi module with extra falgs",
			jibArtifact: latestV2.JibArtifact{
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

func getMavenArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(*latestV2.JibArtifact, string) []string {
	return func(a *latestV2.JibArtifact, minimumVersion string) []string {
		t.CheckDeepEqual(expectedMinimumVersion, minimumVersion)
		if a.Project == "" {
			return []string{"fake-mavenArgs"}
		}
		return []string{"fake-mavenArgs-for-" + a.Project}
	}
}

func getMavenBuildArgsFuncFake(t *testutil.T, expectedMinimumVersion string) func(string, *latestV2.JibArtifact, bool, bool, string) []string {
	return func(goal string, a *latestV2.JibArtifact, skipTests, showColors bool, minimumVersion string) []string {
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
