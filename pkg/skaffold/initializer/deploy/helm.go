/*
Copyright 2022 The Skaffold Authors

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

package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

const (
	nameKey = "name"
)

// for testing
var (
	readFile = os.ReadFile
)

// helm implements deploymentInitializer for the helm deployer.
type helm struct {
	charts []chart
}

type chart struct {
	name        string
	chartValues map[string]interface{}
	path        string
	valueFiles  []string
	version     string
}

// newHelmInitializer returns a helm config generator.
func newHelmInitializer(chartValuesMap map[string][]string) helm {
	var charts []chart
	for chDir, vfs := range chartValuesMap {
		chFile := filepath.Join(chDir, analyze.ChartYaml)
		parsed, err := parseChartValues(chFile)
		if err != nil {
			log.Entry(context.TODO()).Infof("Skipping chart dir %s, as %s could not be parsed as valid yaml", chDir, chFile)
			continue
		}
		charts = append(charts, buildChart(parsed, chDir, vfs))
	}
	return helm{
		charts: charts,
	}
}

// DeployConfig implements the Initializer interface and generates
// a helm configuration
func (h helm) DeployConfig() latest.DeployConfig {
	releases := []latest.HelmRelease{}
	for _, ch := range h.charts {
		// to make skaffold.yaml more portable across OS-es we should always generate /-delimited filePaths
		rPath := strings.ReplaceAll(ch.path, string(os.PathSeparator), "/")
		rVfs := make([]string, len(ch.valueFiles))
		for i, vf := range ch.valueFiles {
			rVfs[i] = strings.ReplaceAll(vf, string(os.PathSeparator), "/")
		}

		r := latest.HelmRelease{
			Name:        ch.name,
			ChartPath:   rPath,
			Version:     ch.version,
			ValuesFiles: rVfs,
		}
		releases = append(releases, r)
	}
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			LegacyHelmDeploy: &latest.LegacyHelmDeploy{
				Releases: releases,
			},
		},
	}
}

func parseChartValues(fp string) (map[string]interface{}, error) {
	in, err := readFile(fp)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if errY := yaml.UnmarshalStrict(in, &m); errY != nil {
		return nil, errY
	}
	return m, nil
}

func getChartName(parsed map[string]interface{}, chDir string) string {
	if v, ok := parsed[nameKey]; ok {
		return v.(string)
	}
	return filepath.Base(chDir)
}

func buildChart(parsed map[string]interface{}, chDir string, vfs []string) chart {
	ch := chart{
		chartValues: parsed,
		name:        getChartName(parsed, chDir),
		path:        chDir,
		valueFiles:  vfs,
	}
	if v := getVersion(parsed); v != "" {
		ch.version = v
	}
	return ch
}

func getVersion(m map[string]interface{}) string {
	if v, ok := m["version"]; ok {
		return v.(string)
	}
	return ""
}
