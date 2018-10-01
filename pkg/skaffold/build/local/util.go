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
	"os"
	"os/exec"
	"path/filepath"
)

func buildMavenCommand(workspace string, args ...string) ([]string, error) {
	maven, err := findBuilder("mvn", "mvnw", workspace)
	if err != nil {
		return nil, err
	}
	return append(maven, args...), nil
}

func buildGradleCommand(workspace string, args ...string) ([]string, error) {
	gradle, err := findBuilder("gradle", "gradlew", workspace)
	if err != nil {
		return nil, err
	}
	return append(gradle, args...), nil
}

// Maven and Gradle projects often provide a wrapper to ensure a particular
// builder version is used.  This function tries to resolve a wrapper
// or otherwise resolves the builder executable.
func findBuilder(builderExecutable string, wrapperScriptName string, workspace string) ([]string, error) {
	wrapperFile := filepath.Join(workspace, wrapperScriptName)
	if isFile(wrapperFile) {
		absolute, err := filepath.Abs(wrapperFile)
		if err != nil {
			return nil, err
		}
		return []string{absolute}, nil
	}
	path, err := exec.LookPath(builderExecutable)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func isFile(filePath string) bool {
	info, error := os.Stat(filePath)
	return error == nil && !info.IsDir()
}
