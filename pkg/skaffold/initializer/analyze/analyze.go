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
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/karrick/godirwalk"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// analyzer is following the visitor pattern. It is called on every file
// as the analysis.analyze function walks the directory structure recursively.
// It can manage state and react to walking events assuming a breadth first search.
type analyzer interface {
	enterDir(dir string)
	analyzeFile(ctx context.Context, file string) error
	exitDir(dir string)
}

type ProjectAnalysis struct {
	configAnalyzer    *skaffoldConfigAnalyzer
	kubeAnalyzer      *kubeAnalyzer
	kustomizeAnalyzer *kustomizeAnalyzer
	helmAnalyzer      *helmAnalyzer
	builderAnalyzer   *builderAnalyzer
	maxFileSize       int64
}

type HelmChartInfo struct {
	chartPaths    map[string][]string
	buildersFound []build.InitBuilder
}

func (h *HelmChartInfo) Charts() map[string][]string {
	return h.chartPaths
}

func (a *ProjectAnalysis) Builders() []build.InitBuilder {
	return a.builderAnalyzer.foundBuilders
}

func (a *ProjectAnalysis) Manifests() []string {
	return a.kubeAnalyzer.kubernetesManifests
}

func (a *ProjectAnalysis) KustomizePaths() []string {
	return a.kustomizeAnalyzer.kustomizePaths
}

func (a *ProjectAnalysis) KustomizeBases() []string {
	return a.kustomizeAnalyzer.bases
}

func (a *ProjectAnalysis) HelmChartInfo() HelmChartInfo {
	return HelmChartInfo{
		chartPaths:    a.helmAnalyzer.chartDirs,
		buildersFound: a.Builders(),
	}
}

func (a *ProjectAnalysis) ChartPaths() map[string][]string {
	return a.helmAnalyzer.chartDirs
}

func (a *ProjectAnalysis) analyzers() []analyzer {
	return []analyzer{
		a.kubeAnalyzer,
		a.kustomizeAnalyzer,
		a.helmAnalyzer,
		a.configAnalyzer,
		a.builderAnalyzer,
	}
}

// NewAnalyzer sets up the analysis of the directory based on the initializer configuration
func NewAnalyzer(c config.Config) *ProjectAnalysis {
	return &ProjectAnalysis{
		kubeAnalyzer:      &kubeAnalyzer{},
		kustomizeAnalyzer: &kustomizeAnalyzer{},
		helmAnalyzer:      &helmAnalyzer{chartDirs: map[string][]string{}},
		builderAnalyzer: &builderAnalyzer{
			findBuilders:         !c.SkipBuild,
			enableJibInit:        c.EnableJibInit,
			enableJibGradleInit:  c.EnableJibGradleInit,
			enableKoInit:         c.EnableKoInit,
			enableBuildpacksInit: c.EnableBuildpacksInit,
			buildpacksBuilder:    c.BuildpacksBuilder,
		},
		configAnalyzer: &skaffoldConfigAnalyzer{
			force:        c.Force,
			analyzeMode:  c.Analyze,
			targetConfig: c.Opts.ConfigurationFile,
		},
		maxFileSize: c.MaxFileSize,
	}
}

// Analyze recursively walks a directory and notifies the analyzers of files and enterDir and exitDir events
// at the end of the analyze function the analysis struct's analyzers should contain the state that we can
// use to do further computation.
func (a *ProjectAnalysis) Analyze(dir string) error {
	for _, analyzer := range a.analyzers() {
		analyzer.enterDir(dir)
	}

	dirents, err := godirwalk.ReadDirents(dir, nil)
	if err != nil {
		return err
	}

	// This is for deterministic results - given the same directory structure
	// init should have the same results.
	sort.Sort(dirents)

	var subdirectories []string

	// Traverse files
	for _, file := range dirents {
		name := file.Name()

		if file.IsDir() {
			if util.IsHiddenDir(name) || skipFolder(name) {
				continue
			}
		} else if util.IsHiddenFile(name) {
			continue
		}

		filePath := filepath.Join(dir, name)

		// If we found a directory, keep track of it until we've gone through all the files first
		if file.IsDir() {
			subdirectories = append(subdirectories, filePath)
			continue
		}

		if a.maxFileSize > 0 {
			stat, err := os.Stat(filePath)
			if err != nil {
				// this is highly unexpected but in case there could be a racey situation where
				// the file gets removed right between ReadDirents and Stat
				continue
			}
			if stat.Size() > a.maxFileSize {
				log.Entry(context.TODO()).Debugf("skipping %s as it is larger (%d) than max allowed size %d", filePath, stat.Size(), a.maxFileSize)
				continue
			}
		}

		// to make skaffold.yaml more portable across OS-es we should always generate /-delimited filePaths
		filePath = strings.ReplaceAll(filePath, string(os.PathSeparator), "/")
		for _, analyzer := range a.analyzers() {
			if err := analyzer.analyzeFile(context.Background(), filePath); err != nil {
				return err
			}
		}
	}

	// Recurse into subdirectories
	for _, subdir := range subdirectories {
		if err := a.Analyze(subdir); err != nil {
			return err
		}
	}

	for _, analyzer := range a.analyzers() {
		analyzer.exitDir(dir)
	}

	return nil
}

func skipFolder(name string) bool {
	return name == "vendor" || name == "node_modules"
}
