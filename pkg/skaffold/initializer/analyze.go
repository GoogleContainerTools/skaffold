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
	kubectlAnalyzer  *kubectlAnalyzer
	skaffoldAnalyzer *skaffoldConfigAnalyzer
	builderAnalyzer  *builderAnalyzer
}

// analyzer is a generic Visitor that is called on every file in the directory
// It can manage state and react to walking events assuming a bread first search
type analyzer interface {
	enterDir(dir string)
	analyzeFile(file string) error
	exitDir(dir string)
}

type directoryAnalyzer struct {
	currentDir string
}

func (a *directoryAnalyzer) analyzeFile(filePath string) error {
	return nil
}

func (a *directoryAnalyzer) enterDir(dir string) {
	a.currentDir = dir
}

func (a *directoryAnalyzer) exitDir(dir string) {
	//pass
}

type kubectlAnalyzer struct {
	directoryAnalyzer
	kubernetesManifests []string
}

func (a *kubectlAnalyzer) analyzeFile(filePath string) error {
	if kubectl.IsKubernetesManifest(filePath) && !IsSkaffoldConfig(filePath) {
		a.kubernetesManifests = append(a.kubernetesManifests, filePath)
	}
	return nil
}

type skaffoldConfigAnalyzer struct {
	directoryAnalyzer
	force bool
}

func (a *skaffoldConfigAnalyzer) analyzeFile(filePath string) error {
	if !IsSkaffoldConfig(filePath) {
		return nil
	}
	if !a.force {
		return fmt.Errorf("pre-existing %s found (you may continue with --force)", filePath)
	}
	logrus.Debugf("%s is a valid skaffold configuration: continuing since --force=true", filePath)
	return nil
}

type builderAnalyzer struct {
	directoryAnalyzer
	enableJibInit       bool
	enableBuildpackInit bool
	findBuilders        bool
	foundBuilders       []InitBuilder

	parentDirToStopFindBuilders string
}

func (a *builderAnalyzer) analyzeFile(filePath string) error {
	if a.findBuilders && (a.parentDirToStopFindBuilders == "" || a.parentDirToStopFindBuilders == a.currentDir) {
		builderConfigs, continueSearchingBuilders := detectBuilders(a.enableJibInit, a.enableBuildpackInit, filePath)
		a.foundBuilders = append(a.foundBuilders, builderConfigs...)
		if !continueSearchingBuilders {
			a.parentDirToStopFindBuilders = a.currentDir
		}
	}
	return nil
}

func (a *builderAnalyzer) exitDir(dir string) {
	if a.parentDirToStopFindBuilders == dir {
		a.parentDirToStopFindBuilders = ""
	}
}

// analyze recursively walks a directory and returns the k8s configs and builder configs that it finds
func (a *analysis) analyze(dir string) error {
	for _, analyzer := range a.analyzers() {
		analyzer.enterDir(dir)
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
			if err := analyzer.analyzeFile(filePath); err != nil {
				return err
			}
		}
	}

	// Recurse into subdirectories
	for _, subdir := range subdirectories {
		if err = a.analyze(filepath.Join(dir, subdir.Name())); err != nil {
			return err
		}
	}

	for _, analyzer := range a.analyzers() {
		analyzer.exitDir(dir)
	}
	return nil
}

func (a *analysis) analyzers() []analyzer {
	return []analyzer{
		a.kubectlAnalyzer,
		a.skaffoldAnalyzer,
		a.builderAnalyzer,
	}
}
