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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependenciesMaven(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"mvn jib:_skaffold-files -q",
		"dep1\ndep2\n\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("pom.xml", "")

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesMavenOnWindows(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"mvn jib:_skaffold-files -q",
		"\n\ndep1\ndep2",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("pom.xml", "")

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, true)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesMavenWithWrapper(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"./mvnw jib:_skaffold-files -q",
		"\ndep1\ndep2\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("pom.xml", "")
	tmpDir.Write("mvnw", "")

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesMavenWithWrapperOnWindows(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"call mvnw.cmd jib:_skaffold-files -q",
		"\n\ndep1\ndep2\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("pom.xml", "")
	tmpDir.Write("mvnw.cmd", "")

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, true)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesMavenNoPomXml(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(
		"ignored",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	_, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{}, false)

	if err.Error() != "no pom.xml found" {
		t.Errorf("Unexpected error message %s", err.Error())
	}
}

func TestGetDependenciesGradle(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"gradle _jibSkaffoldFiles -q",
		"dep1\ndep2\n\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("build.gradle", "")

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesGradleOnWindows(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"gradle _jibSkaffoldFiles -q",
		"\n\ndep1\ndep2",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("build.gradle", "")

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, true)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesGradleWithWrapper(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"./gradlew _jibSkaffoldFiles -q",
		"\ndep1\ndep2\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("build.gradle", "")
	tmpDir.Write("gradlew", "")

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, false)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesGradleWithWrapperOnWindows(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmdOut(
		"call gradlew.bat _jibSkaffoldFiles -q",
		"\n\ndep1\ndep2\n\n",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("build.gradle", "")
	tmpDir.Write("gradlew.bat", "")

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, true)

	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"dep1", "dep2"}, deps)
}

func TestGetDependenciesGradleNoPomXml(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(
		"ignored",
		nil,
	)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	_, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{}, false)

	if err.Error() != "no build.gradle found" {
		t.Errorf("Unexpected error message %s", err.Error())
	}
}
