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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
)

// kustomizeAnalyzer is a Visitor during the directory analysis that finds kustomize files
type kustomizeAnalyzer struct {
	directoryAnalyzer

	bases          []string
	kustomizePaths []string
}

func (k *kustomizeAnalyzer) analyzeFile(path string) error {
	switch {
	case deploy.IsKustomizationBase(path):
		k.bases = append(k.bases, filepath.Dir(path))
	case deploy.IsKustomizationPath(path):
		k.kustomizePaths = append(k.kustomizePaths, filepath.Dir(path))
	}
	return nil
}
