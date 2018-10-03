// +build linux darwin

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

func getWrapper(defaultExecutable string) string {
	switch defaultExecutable {
	case gradleExecutable:
		return "gradlew"
	case mavenExecutable:
		return "mvnw"
	}
	return defaultExecutable
}

func getCommand(workspace string, defaultExecutable string, defaultSubCommand []string) (executable string, subCommand []string) {
	executable = defaultExecutable
	subCommand = defaultSubCommand

	if wrapperExecutable, err := resolveFile(workspace, getWrapper(defaultExecutable)); err == nil {
		executable = wrapperExecutable
	}

	return executable, subCommand
}
