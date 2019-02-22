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

package util

var SkipWrapperCheck = false

// CommandWrapper defines an association between an executable command (like `gradle`)
// and possible command wrappers (like `gradlew`).  `CreateCommand` uses this definition
// to create a `Cmd` object.  Maven and Gradle projects often provide a wrapper script
// to ensure a particular version of their builder is used.
type CommandWrapper struct {
	// Executable is the base name of the command, like `gradle`
	Executable string

	// Wrapper is the optional base name of a command wrapper, like `gradlew`
	Wrapper string
}
