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
	"path/filepath"
	"strings"

	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
)

// helmAnalyzer is a Visitor during the directory analysis that finds helm charts
type helmAnalyzer struct {
	directoryAnalyzer
	chartDirs map[string][]string
}

func (h *helmAnalyzer) analyzeFile(ctx context.Context, filePath string) error {
	if deploy.IsHelmChart(filePath) {
		chDir, _ := filepath.Split(filePath)
		h.chartDirs[filepath.Clean(chDir)] = []string{}
		return nil
	}
	if strings.HasSuffix(filePath, "yaml") || strings.HasSuffix(filePath, "yml") {
		dir, _ := filepath.Split(filePath)
		if s, ok := h.chartDirs[filepath.Clean(dir)]; ok {
			h.chartDirs[filepath.Clean(dir)] = append(s, filePath)
		}
	}
	return nil
}
