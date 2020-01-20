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

package initializer

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/karrick/godirwalk"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type analysis struct {
	kubectlAnalyzer  *KubectlAnalyzer
	skaffoldAnalyzer *SkaffoldConfigAnalyzer
	builderAnalyzer  *BuilderAnalyzer
}

// Analyzer is a generic Visitor that is called on every file in the directory
// It can manage state and react to walking events assuming a bread first search
type Analyzer interface {
	EnterDir(dir string)
	AnalyzeFile(file string) error
	ExitDir(dir string)
}

type DirectoryAnalyzer struct {
	currentDir string
}

func (a *DirectoryAnalyzer) AnalyzeFile(filePath string) error {
	return nil
}

func (a *DirectoryAnalyzer) EnterDir(dir string) {
	a.currentDir = dir
}
func (a *DirectoryAnalyzer) ExitDir(dir string) {
	//pass
}

type KubectlAnalyzer struct {
	DirectoryAnalyzer
	kubernetesManifests []string
}

func (a *KubectlAnalyzer) AnalyzeFile(filePath string) error {
	isSkaffoldConfig := IsSkaffoldConfig(filePath)
	isKubernetesManifest := kubectl.IsKubernetesManifest(filePath)

	if isKubernetesManifest && !isSkaffoldConfig {
		a.kubernetesManifests = append(a.kubernetesManifests, filePath)
	}
	return nil
}

type SkaffoldConfigAnalyzer struct {
	DirectoryAnalyzer
	force bool
}

func (a *SkaffoldConfigAnalyzer) AnalyzeFile(filePath string) error {
	isSkaffoldConfig := IsSkaffoldConfig(filePath)
	if isSkaffoldConfig {
		if !a.force {
			return fmt.Errorf("pre-existing %s found (you may continue with --force)", filePath)
		}
		logrus.Debugf("%s is a valid skaffold configuration: continuing since --force=true", filePath)
	}
	return nil
}

type BuilderAnalyzer struct {
	DirectoryAnalyzer
	enableJibInit       bool
	enableBuildpackInit bool
	findBuilders        bool
	foundBuilders       []InitBuilder

	parentDirToStopFindBuilders string
}

func (a *BuilderAnalyzer) AnalyzeFile(filePath string) error {
	if a.findBuilders && (a.parentDirToStopFindBuilders == "" || a.parentDirToStopFindBuilders == a.currentDir) {
		builderConfigs, continueSearchingBuilders := detectBuilders(a.enableJibInit, a.enableBuildpackInit, filePath)
		a.foundBuilders = append(a.foundBuilders, builderConfigs...)
		if !continueSearchingBuilders {
			a.parentDirToStopFindBuilders = a.currentDir
		}
	}
	return nil
}

func (a *BuilderAnalyzer) ExitDir(dir string) {
	if a.parentDirToStopFindBuilders == dir {
		a.parentDirToStopFindBuilders = ""
	}
}

// walk recursively walks a directory and returns the k8s configs and builder configs that it finds
func (a *analysis) walk(root string) error {
	var analyze func(dir string) error
	analyze = func(dir string) error {
		for _, analyzer := range a.analyzers() {
			analyzer.EnterDir(dir)
		}
		dirents, err := godirwalk.ReadDirents(dir, nil)
		if err != nil {
			return err
		}

		var subdirectories []*godirwalk.Dirent
		//this is for deterministic results - given the same directory structure
		//init should have the same results
		sort.Sort(dirents)

		// Traverse files
		for _, file := range dirents {
			if util.IsHiddenFile(file.Name()) || util.IsHiddenDir(file.Name()) {
				continue
			}

			// If we found a directory, keep track of it until we've gone through all the files first
			if file.IsDir() {
				subdirectories = append(subdirectories, file)
				continue
			}

			filePath := filepath.Join(dir, file.Name())
			for _, analyzer := range a.analyzers() {
				if err := analyzer.AnalyzeFile(filePath); err != nil {
					return err
				}
			}
		}

		// Recurse into subdirectories
		for _, subdir := range subdirectories {
			if err = analyze(filepath.Join(dir, subdir.Name())); err != nil {
				return err
			}
		}

		for _, analyzer := range a.analyzers() {
			analyzer.ExitDir(dir)
		}
		return nil
	}

	return analyze(root)
}

func (a *analysis) analyzers() []Analyzer {
	return []Analyzer{
		a.kubectlAnalyzer,
		a.skaffoldAnalyzer,
		a.builderAnalyzer,
	}
}
