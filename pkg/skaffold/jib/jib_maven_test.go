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

func TestMavenWrapperDefinition(t *testing.T) {
	testutil.CheckDeepEqual(t, "mvn", MavenCommand.Executable)
	testutil.CheckDeepEqual(t, "mvnw", MavenCommand.Wrapper)
}

func TestGetDependenciesMaven(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("build", "")
	tmpDir.Write("dep1", "")
	tmpDir.Write("dep2", "")

	build := tmpDir.Path("build")
	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dep2")

	ctx := context.Background()

	var tests = []struct {
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
		t.Run(test.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOutErr(
				strings.Join(getCommandMaven(ctx, tmpDir.Root(), &latest.JibMavenArtifact{Module: "maven-test"}).Args, " "),
				test.stdout,
				test.err,
			)

			// Change build file mod time
			os.Chtimes(build, test.modTime, test.modTime)

			deps, err := GetDependenciesMaven(ctx, tmpDir.Root(), &latest.JibMavenArtifact{Module: "maven-test"})
			if test.err != nil {
				testutil.CheckErrorAndDeepEqual(t, true, err, "getting jibMaven dependencies: initial Jib dependency refresh failed: failed to get Jib dependencies; it's possible you are using an old version of Jib (Skaffold requires Jib v1.0.2+): "+test.err.Error(), err.Error())
			} else {
				testutil.CheckDeepEqual(t, test.expected, deps)
			}
		})
	}
}

func TestGetCommandMaven(t *testing.T) {
	ctx := context.Background()
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
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description: "maven with extra flags",
			jibMavenArtifact: latest.JibMavenArtifact{
				Flags: []string{"-DskipTests", "-x"},
			},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"-DskipTests", "-x", "--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description:      "maven with profile",
			jibMavenArtifact: latest.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace: []string{},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--activate-profiles", "profile", "--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description:      "maven with wrapper no profile",
			jibMavenArtifact: latest.JibMavenArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description:      "maven with wrapper no profile",
			jibMavenArtifact: latest.JibMavenArtifact{},
			filesInWorkspace: []string{"mvnw", "mvnw.cmd"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description:      "maven with wrapper and profile",
			jibMavenArtifact: latest.JibMavenArtifact{Profile: "profile"},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--activate-profiles", "profile", "--non-recursive", "jib:_skaffold-files-v2", "--quiet"})
			},
		},
		{
			description:      "maven with multi-modules",
			jibMavenArtifact: latest.JibMavenArtifact{Module: "module"},
			filesInWorkspace: []string{"mvnw", "mvnw.bat"},
			expectedCmd: func(workspace string) *exec.Cmd {
				return MavenCommand.CreateCommand(ctx, workspace, []string{"--projects", "module", "--also-make", "jib:_skaffold-files-v2", "--quiet"})
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

			cmd := getCommandMaven(ctx, tmpDir.Root(), &test.jibMavenArtifact)
			expectedCmd := test.expectedCmd(tmpDir.Root())
			testutil.CheckDeepEqual(t, expectedCmd.Path, cmd.Path)
			testutil.CheckDeepEqual(t, expectedCmd.Args, cmd.Args)
			testutil.CheckDeepEqual(t, expectedCmd.Dir, cmd.Dir)
		})
	}
}

func TestGenerateMavenArgs(t *testing.T) {
	var testCases = []struct {
		in        latest.JibMavenArtifact
		skipTests bool
		out       []string
	}{
		{latest.JibMavenArtifact{}, false, []string{"-Djib.console=plain", "--non-recursive", "prepare-package", "jib:goal", "-Dimage=image"}},
		{latest.JibMavenArtifact{}, true, []string{"-Djib.console=plain", "--non-recursive", "-DskipTests=true", "prepare-package", "jib:goal", "-Dimage=image"}},
		{latest.JibMavenArtifact{Profile: "profile"}, false, []string{"-Djib.console=plain", "--activate-profiles", "profile", "--non-recursive", "prepare-package", "jib:goal", "-Dimage=image"}},
		{latest.JibMavenArtifact{Profile: "profile"}, true, []string{"-Djib.console=plain", "--activate-profiles", "profile", "--non-recursive", "-DskipTests=true", "prepare-package", "jib:goal", "-Dimage=image"}},
		{latest.JibMavenArtifact{Module: "module"}, false, []string{"-Djib.console=plain", "--projects", "module", "--also-make", "package", "-Dimage=image"}},
		{latest.JibMavenArtifact{Module: "module"}, true, []string{"-Djib.console=plain", "--projects", "module", "--also-make", "-DskipTests=true", "package", "-Dimage=image"}},
		{latest.JibMavenArtifact{Module: "module", Profile: "profile"}, false, []string{"-Djib.console=plain", "--activate-profiles", "profile", "--projects", "module", "--also-make", "package", "-Dimage=image"}},
		{latest.JibMavenArtifact{Module: "module", Profile: "profile"}, true, []string{"-Djib.console=plain", "--activate-profiles", "profile", "--projects", "module", "--also-make", "-DskipTests=true", "package", "-Dimage=image"}},
	}

	for _, tt := range testCases {
		args := GenerateMavenArgs("goal", "image", &tt.in, tt.skipTests)

		testutil.CheckDeepEqual(t, tt.out, args)
	}
}
