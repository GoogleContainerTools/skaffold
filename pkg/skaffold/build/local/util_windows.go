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
	"runtime"
)

// Maven and Gradle projects often provide a wrapper to ensure a particular
// builder version is used.  This function tries to resolve a wrapper
// or otherwise resolves the builder executable.
func findBuilder(builderExecutable string, wrapperScriptName string, workspace string) ([]string, error) {
	wrapperFile := filepath.Join(workspace, wrapperScriptName)
	if isFile(wrapperFile) {
		return filepath.Abs(wrapperFile)
	}
	if cmdFile := wrapperFile + ".cmd"; isFile(cmdFile) {
		path, err := filepath.Abs(cmdFile)
		if err != nil {
			return nil, err
		}
		return []string{"cmd", "/c", path}, nil
	}
	if batFile := wrapperFile + ".bat"; isFile(batFile) {
		path, err := filepath.Abs(batFile)
		if err != nil {
			return nil, err
		}
		return []string{"cmd", "/c", path}, nil
	}
	path, err := exec.LookPath(builderExecutable)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
