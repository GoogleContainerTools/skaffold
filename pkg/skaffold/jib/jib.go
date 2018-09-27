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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetDependenciesMaven finds the source dependencies for the given jib-maven artifact.
// All paths are absolute.
// TODO(coollog): Add support for multi-module projects.
func GetDependenciesMaven(workspace string, _ /*a*/ *v1alpha3.JibMavenArtifact, isWindows bool) ([]string, error) {
	return getDependencies(workspace, "pom.xml", "mvn", "mvnw.cmd", isWindows, []string{"jib:_skaffold-files", "-q"}, "jib-maven")
}

func GetDependenciesGradle(workspace string, _ /*a*/ *v1alpha3.JibGradleArtifact, isWindows bool) ([]string, error) {
	return getDependencies(workspace, "build.gradle", "gradle", "gradlew.bat", isWindows, []string{"_jibSkaffoldFiles", "-q"}, "jib-gradle")
}

func exists(workspace string, filename string) bool {
	_, err := os.Stat(filepath.Join(workspace, filename))
	return err == nil
}

func getDependencies(workspace string, buildFile string, defaultExecutable string, windowsExecutable string, isWindows bool, subCommand []string, artifactName string) ([]string, error) {
	if !exists(workspace, buildFile) {
		return nil, errors.Errorf("no %s found", buildFile)
	}

	executable := defaultExecutable
	if isWindows {
		if exists(workspace, windowsExecutable) {
			executable = "call"
			subCommand = append([]string{windowsExecutable}, subCommand...)
		}
	} else {
		wrapperExecutable := defaultExecutable + "w"
		if exists(workspace, wrapperExecutable) {
			executable = "./" + wrapperExecutable
		}
	}

	cmd := exec.Command(executable, subCommand...)
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s dependencies", artifactName)
	}

	lines := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range lines {
		if l == "" {
			continue
		}
		deps = append(deps, l)
	}
	return deps, nil
}
