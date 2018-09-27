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
	if !exists(workspace, "pom.xml") {
		return nil, errors.New("no pom.xml found")
	}

	mavenSubcommand := []string{"jib:_skaffold-files", "-q"}

	mavenExecutable := "mvn"
	if isWindows {
		if exists(workspace, "mvnw.cmd") {
			mavenExecutable = "call"
			mavenSubcommand = append([]string{"mvnw.cmd"}, mavenSubcommand...)
		}
	} else if exists(workspace, "mvnw") {
		mavenExecutable = "./mvnw"
	}

	cmd := exec.Command(mavenExecutable, mavenSubcommand...)
	cmd.Dir = workspace
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting jib-maven dependencies")
	}

	deps := strings.Split(string(stdout), "\n")
	return deps, nil
}

func GetDependenciesGradle(_ /*workspace*/ string, _ /*a*/ *v1alpha3.JibGradleArtifact) ([]string, error) {
	return nil, errors.New("jib gradle support is unimplemented")
}

func exists(workspace string, filename string) bool {
	_, err := os.Stat(filepath.Join(workspace, filename))
	return err == nil
}
