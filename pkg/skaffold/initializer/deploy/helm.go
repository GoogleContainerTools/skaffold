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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// helm implements deploymentInitializer for the helm deployer.
type helm struct {
	charts []chart
}

type chart struct {
	name      string
	path      string
	valueFile string
	overrides map[string]string
}

// newHelmInitializer returns a helm config generator.
func newHelmInitializer(chartTemplatesMap map[string][]string, builders []build.InitBuilder) helm {
	var charts []chart

	for chDir, _ := range chartTemplatesMap {
		// find value files
		vf := findValuesFile(chDir)
		// generate the artifactsOverride key
		charts = append(charts, chart{
			name:      resolveChartName(chDir),
			path:      chDir,
			valueFile: vf,
		})
	}
	updated := make([]chart, len(charts))
	return helm{
		charts: updated,
	}
}

// DeployConfig implements the Initializer interface and generates
// a helm configuration
func (h helm) DeployConfig() (latest.DeployConfig, []latest.Profile) {
	releases := []latest.HelmRelease{}
	for _, ch := range h.charts {
		chDir, _ := filepath.Split(ch.path)
		releases = append(releases, latest.HelmRelease{
			Name:        ch.name,
			ChartPath:   chDir,
			ValuesFiles: []string{filepath.Join(chDir, "values.yaml")},
		})

	}
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			HelmDeploy: &latest.HelmDeploy{
				Releases: releases,
			},
		},
	}, nil
}

// Validate implements the Initializer interface and ensures
// we have at least one manifest before generating a config
func (h helm) Validate() error {
	if len(h.charts) == 0 {
		return errors.NoHelmChartsErr{}
	}
	return nil
}

// we don't generate manifests for helm
func (h helm) AddManifestForImage(string, string) {}

func resolveChartName(chDirPath string) string {
	_, chDirName := filepath.Split(filepath.Clean(chDirPath))
	if chDirName == "charts" {
		return "chart-foo"
	}
	return chDirName
}
