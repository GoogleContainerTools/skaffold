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
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependencies(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	tmpDir.Touch("dep1", "dep2", "dep3/fileA", "dep3/sub/path/fileB")
	dep1 := tmpDir.Path("dep1")
	dep2 := tmpDir.Path("dep2")
	dep3 := tmpDir.Path("dep3")

	tests := []struct {
		description  string
		stdout       string
		shouldErr    bool
		expectedDeps []string
	}{
		{
			description:  "empty",
			stdout:       "",
			shouldErr:    true,
			expectedDeps: nil,
		},
		{
			description:  "base case",
			stdout:       "BEGIN JIB JSON\n{\"build\":[],\"inputs\":[],\"ignore\":[]}",
			shouldErr:    false,
			expectedDeps: nil,
		},
		{
			description:  "file in build and file in input",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep1, dep2),
			shouldErr:    false,
			expectedDeps: []string{"dep1", "dep2"},
		},
		{
			description:  "dir in input should be expanded",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\"],\"ignore\":[]}\n", dep3),
			shouldErr:    false,
			expectedDeps: []string{filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			description:  "files and dir in input should be expanded",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			shouldErr:    false,
			expectedDeps: []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			description:  "non-existing files should be ignored",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\",\"nonexistent\",\"%s\"],\"ignore\":[]}\n", dep1, dep2, dep3),
			shouldErr:    false,
			expectedDeps: []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA"), filepath.FromSlash("dep3/sub/path/fileB")},
		},
		{
			description:  "ignored files should not be reported",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[],\"inputs\":[\"%s\",\"%s\"],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep2),
			shouldErr:    false,
			expectedDeps: []string{"dep1"},
		},
		{
			description:  "ignored directories should not be reported",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\"],\"inputs\":[\"%s\"],\"ignore\":[\"%s\",\"%s\"]}\n", dep1, dep3, dep1, dep3),
			shouldErr:    false,
			expectedDeps: nil,
		},
		{
			description:  "partial subpaths should be ignored",
			stdout:       fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\",\"%s\",\"%s\"],\"inputs\":[],\"ignore\":[\"%s\"]}\n", dep1, dep2, dep3, tmpDir.Path("dep3/sub/path")),
			shouldErr:    false,
			expectedDeps: []string{"dep1", "dep2", filepath.FromSlash("dep3/fileA")},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				"ignored",
				test.stdout,
			))

			results, err := getDependencies(tmpDir.Root(), exec.Cmd{Args: []string{"ignored"}, Dir: tmpDir.Root()}, &latest.JibArtifact{Project: util.RandomID()})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDeps, results)
		})
	}
}

func TestGetUpdatedDependencies(t *testing.T) {
	testutil.Run(t, "Both build definitions are created at the same time", func(t *testutil.T) {
		tmpDir := t.NewTempDir()

		stdout := fmt.Sprintf("BEGIN JIB JSON\n{\"build\":[\"%s\",\"%s\"],\"inputs\":[],\"ignore\":[]}\n", tmpDir.Path("build.gradle"), tmpDir.Path("settings.gradle"))
		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("ignored", stdout).
			AndRunOut("ignored", stdout).
			AndRunOut("ignored", stdout),
		)

		listCmd := exec.Cmd{Args: []string{"ignored"}, Dir: tmpDir.Root()}
		artifact := &latest.JibArtifact{Project: util.RandomID()}

		// List dependencies
		_, err := getDependencies(tmpDir.Root(), listCmd, artifact)
		t.CheckNoError(err)

		// Create new build definition files
		tmpDir.
			Write("build.gradle", "").
			Write("settings.gradle", "")

		// Update dependencies
		_, err = getDependencies(tmpDir.Root(), listCmd, artifact)
		t.CheckNoError(err)
	})
}

func TestPluginName(t *testing.T) {
	testutil.CheckDeepEqual(t, "Jib Maven Plugin", PluginName(JibMaven))
	testutil.CheckDeepEqual(t, "Jib Gradle Plugin", PluginName(JibGradle))
}

func TestPluginType_IsKnown(t *testing.T) {
	tests := []struct {
		value PluginType
		known bool
	}{
		{JibMaven, true},
		{JibGradle, true},
		{PluginType(0), false},
		{PluginType(-1), false},
		{PluginType(3), false},
	}
	for _, test := range tests {
		testutil.Run(t, string(test.value), func(t *testutil.T) {
			t.CheckDeepEqual(test.known, test.value.IsKnown())
		})
	}
}

func TestDeterminePluginType(t *testing.T) {
	tests := []struct {
		description string
		files       []string
		artifact    *latest.JibArtifact
		shouldErr   bool
		PluginType  PluginType
	}{
		{"empty", []string{}, nil, true, PluginType("")},
		{"gradle-2", []string{"gradle.properties"}, nil, false, JibGradle},
		{"gradle-3", []string{"gradlew"}, nil, false, JibGradle},
		{"gradle-4", []string{"gradlew.bat"}, nil, false, JibGradle},
		{"gradle-5", []string{"gradlew.cmd"}, nil, false, JibGradle},
		{"gradle-6", []string{"settings.gradle"}, nil, false, JibGradle},
		{"gradle-kotlin-1", []string{"build.gradle.kts"}, nil, false, JibGradle},
		{"maven-1", []string{"pom.xml"}, nil, false, JibMaven},
		{"maven-2", []string{".mvn/maven.config"}, nil, false, JibMaven},
		{"maven-3", []string{".mvn/extensions.xml"}, nil, false, JibMaven},
		{"gradle override", []string{"pom.xml"}, &latest.JibArtifact{Type: string(JibGradle)}, false, JibGradle},
		{"maven override", []string{"build.gradle"}, &latest.JibArtifact{Type: string(JibMaven)}, false, JibMaven},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			buildDir := t.NewTempDir()
			buildDir.Touch(test.files...)
			PluginType, err := DeterminePluginType(buildDir.Root(), test.artifact)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.PluginType, PluginType)
		})
	}
}

func TestGetProjectKey(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.JibArtifact
		workspace   string
		expected    projectKey
	}{
		{
			"empty project",
			&latest.JibArtifact{},
			"dir",
			projectKey("dir+"),
		},
		{
			"non-empty project",
			&latest.JibArtifact{Project: "project"},
			"dir",
			projectKey("dir+project"),
		},
	}
	for _, test := range tests {
		projectKey := getProjectKey(test.workspace, test.artifact)
		testutil.CheckDeepEqual(t, test.expected, projectKey)
	}
}
