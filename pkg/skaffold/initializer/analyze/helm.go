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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

const (
	ChartYaml = "Chart.yaml"
)

// helmAnalyzer is a Visitor during the directory analysis that finds helm charts
type helmAnalyzer struct {
	directoryAnalyzer
	chartDirs map[string][]string
}

func (h *helmAnalyzer) analyzeFile(ctx context.Context, fp string) error {
	if isHelmChart(fp) {
		chDir, _ := filepath.Split(fp)
		h.chartDirs[filepath.Clean(chDir)] = []string{}
		return nil
	}
	if isValueFile(fp) {
		dir, _ := filepath.Split(fp)
		dir = filepath.Clean(dir)
		if s, ok := h.chartDirs[dir]; ok {
			h.chartDirs[dir] = append(s, fp)
		} else {
			if hasChart(dir) {
				h.chartDirs[dir] = []string{fp}
			}
			log.Entry(context.TODO()).Debugf("ignoring a yaml file %s not part of any chart ", fp)
		}
	}
	return nil
}

func isValueFile(fp string) bool {
	return strings.HasSuffix(fp, "yaml") || strings.HasSuffix(fp, "yml")
}

func isHelmChart(path string) bool {
	return filepath.Base(path) == ChartYaml
}

func hasChart(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ChartYaml)); os.IsNotExist(err) {
		return false
	}
	return true
}
