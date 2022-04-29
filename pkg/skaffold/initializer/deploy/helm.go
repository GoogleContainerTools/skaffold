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
	"io/ioutil"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/sirupsen/logrus"
)

const (
	nameKey = "name"
)

// helm implements deploymentInitializer for the helm deployer.
type helm struct {
	charts []chart
}

type chart struct {
	name        string
	chartValues map[string]string
	path        string
	valueFiles  []string
}

// newHelmInitializer returns a helm config generator.
func newHelmInitializer(chartValuesMap map[string][]string) helm {
	var charts []chart
	for chDir, vfs := range chartValuesMap {
		chFile := filepath.Join(chDir, analyze.ChartYaml)
		parsed, err := parseChartValues(chFile)
		if err != nil {
			logrus.Infof("Skipping chart dir %s, as %s could not be parsed as valid yaml", chDir, chFile)
			continue
		}
		name := getChartName(parsed, chDir)
		charts = append(charts, chart{
			chartValues: parsed,
			name:        name,
			path:        chDir,
			valueFiles:  vfs,
		})
	}
	return helm{
		charts: charts,
	}
}

// DeployConfig implements the Initializer interface and generates
// a helm configuration
func (h helm) DeployConfig() (latest.DeployConfig, []latest.Profile) {
	releases := []latest.HelmRelease{}
	for _, ch := range h.charts {
		releases = append(releases, latest.HelmRelease{
			Name:        ch.name,
			ChartPath:   ch.path,
			ValuesFiles: ch.valueFiles,
		})

	}
	return latest.DeployConfig{
		DeployType: latest.DeployType{
			LegacyHelmDeploy: &latest.LegacyHelmDeploy{
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

// GetImages return an empty string for helm.
func (h helm) GetImages() []string {
	artifacts := []string{}
	for _, ch := range h.charts {
		artifacts = append(artifacts, ch.name)
	}
	return artifacts
}

func parseChartValues(fp string) (map[string]string, error) {
	in, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if err := yaml.UnmarshalStrict(in, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func getChartName(parsed map[string]string, chDir string) string {
	if v, ok := parsed[nameKey]; ok {
		return v
	}
	return filepath.Base(chDir)
}
