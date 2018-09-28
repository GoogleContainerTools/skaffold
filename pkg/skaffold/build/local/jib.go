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

package local

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/pkg/errors"
)

func (b *Builder) buildJibMaven(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibMavenArtifact, imageName string) (string, error) {
	maven, err := findBuilder("mvn", "mvnw", workspace)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, maven, "prepare-package", "jib:build", "-Dimage="+imageName)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	return imageName, nil
}

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, workspace string, a *v1alpha3.JibGradleArtifact, imageName string) (string, error) {
	gradle, err := findBuilder("gradle", "gradlew", workspace)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, gradle, ":jib", "--image="+imageName)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	return imageName, nil
}

// Maven and Gradle projects often provide a wrapper to ensure a particular
// builder version is used.  This function tries to resolve a wrapper
// or otherwise resolves the builder executable.
func findBuilder(builderExecutable string, wrapperScriptName string, workspace string) (string, error) {
	wrapperFile := filepath.Join(workspace, wrapperScriptName)
	info, error := os.Stat(wrapperFile)
	if error == nil && !info.IsDir() {
		return filepath.Abs(wrapperFile)
	}
	if runtime.GOOS == "windows" {
		batFile := wrapperFile + ".cmd"
		info, error := os.Stat(batFile)
		if error == nil && !info.IsDir() {
			return filepath.Abs(batFile)
		}
		cmdFile := wrapperFile + ".cmd"
		info, error = os.Stat(cmdFile)
		if error == nil && !info.IsDir() {
			return filepath.Abs(cmdFile)
		}
	}
	return exec.LookPath(builderExecutable)
}
