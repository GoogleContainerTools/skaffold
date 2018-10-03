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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/pkg/errors"
)

// GetDependenciesMaven finds the source dependencies for the given jib-maven artifact.
// All paths are absolute.
// TODO(coollog): Add support for multi-module projects.
func GetDependenciesMaven(workspace string, a *v1alpha3.JibMavenArtifact) ([]string, error) {
	executable, subCommand := getCommandMaven(workspace, a)
	return getDependencies(workspace, "pom.xml", executable, subCommand, "jib-maven")
}

// GetDependenciesGradle finds the source dependencies for the given jib-gradle artifact.
// All paths are absolute.
func GetDependenciesGradle(workspace string, a *v1alpha3.JibGradleArtifact) ([]string, error) {
	executable, subCommand := getCommandGradle(workspace, a)
	return getDependencies(workspace, "build.gradle", executable, subCommand, "jib-gradle")
}

// resolveFile resolves the absolute path of the file named filename in directory workspace, erroring if it is not a file
func resolveFile(workspace string, filename string) (string, error) {
	file := filepath.Join(workspace, filename)
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.Errorf("%s is a directory", file)
	}
	return filepath.Abs(file)
}

func getDependencies(workspace string, buildFile string, executable string, subCommand []string, artifactName string) ([]string, error) {
	if _, err := resolveFile(workspace, buildFile); err != nil {
		return nil, errors.Errorf("no %s found", buildFile)
	}

	cmd := exec.Command(executable, subCommand...)
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s dependencies", artifactName)
	}

	return getDepsFromStdout(string(stdout)), nil
}

const (
	mavenExecutable  = "mvn"
	gradleExecutable = "gradle"
)

func getCommandMaven(workspace string, a *v1alpha3.JibMavenArtifact) (executable string, subCommand []string) {
	subCommand = []string{"jib:_skaffold-files", "-q"}
	if a.Profile != "" {
		subCommand = append(subCommand, "-P", a.Profile)
	}

	return getCommand(workspace, mavenExecutable, subCommand)
}

func getCommandGradle(workspace string, _ /* a */ *v1alpha3.JibGradleArtifact) (executable string, subCommand []string) {
	return getCommand(workspace, gradleExecutable, []string{"_jibSkaffoldFiles", "-q"})
}

func getDepsFromStdout(stdout string) []string {
	lines := strings.Split(stdout, "\n")
	var deps []string
	for _, l := range lines {
		if l == "" {
			continue
		}
		deps = append(deps, l)
	}
	return deps
}
