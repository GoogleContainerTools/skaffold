/*
Copyright 2020 The Skaffold Authors

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

package analyze

import (
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
)

type builderAnalyzer struct {
	directoryAnalyzer
	enableJibInit        bool
	enableJibGradleInit  bool
	enableBuildpacksInit bool
	findBuilders         bool
	buildpacksBuilder    string
	foundBuilders        []build.InitBuilder

	parentDirToStopFindJibSettings string
}

func (a *builderAnalyzer) analyzeFile(filePath string) error {
	if a.findBuilders {
		lookForJib := a.parentDirToStopFindJibSettings == "" || a.parentDirToStopFindJibSettings == a.currentDir
		builderConfigs, lookForJib := a.detectBuilders(filePath, lookForJib)
		a.foundBuilders = append(a.foundBuilders, builderConfigs...)
		if !lookForJib {
			a.parentDirToStopFindJibSettings = a.currentDir
		}
	}
	return nil
}

func (a *builderAnalyzer) exitDir(dir string) {
	if a.parentDirToStopFindJibSettings == dir {
		a.parentDirToStopFindJibSettings = ""
	}
}

// detectBuilders checks if a path is a builder config, and if it is, returns the InitBuilders representing the
// configs. Also returns a boolean marking search completion for subdirectories (true = subdirectories should
// continue to be searched, false = subdirectories should not be searched for more builders)
func (a *builderAnalyzer) detectBuilders(path string, detectJib bool) ([]build.InitBuilder, bool) {
	var results []build.InitBuilder
	searchSubDirectories := true

	// TODO: Remove backwards compatibility if statement (not entire block)
	if a.enableJibInit && detectJib {
		// Check for jib
		if builders := jib.Validate(path, a.enableJibGradleInit); builders != nil {
			for i := range builders {
				results = append(results, builders[i])
			}
			searchSubDirectories = false
		}
	}

	// Check for Dockerfile
	base := filepath.Base(path)
	if strings.Contains(strings.ToLower(base), "dockerfile") {
		if docker.Validate(path) {
			results = append(results, docker.ArtifactConfig{
				File: path,
			})
		}
	}

	// TODO: Remove backwards compatibility if statement (not entire block)
	if a.enableBuildpacksInit {
		// Check for buildpacks
		if buildpacks.Validate(path) {
			results = append(results, buildpacks.ArtifactConfig{
				File:    path,
				Builder: a.buildpacksBuilder,
			})
		}
	}

	return results, searchSubDirectories
}
