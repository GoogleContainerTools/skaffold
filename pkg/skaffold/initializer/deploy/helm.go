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
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

const (
	nameKey = "name"
)

// for testing
var (
	readFile = ioutil.ReadFile
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
	repo        string
	version     string
	isRemote    bool
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
		remotes := getRemoteChart(parsed)
		charts = append(charts, buildChart(parsed, chDir, vfs))
		charts = append(charts, remotes...)
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
		var r latest.HelmRelease
		if ch.isRemote {
			r = latest.HelmRelease{
				Name:        ch.name,
				Repo:        ch.repo,
				Version:     ch.version,
				RemoteChart: ch.name,
			}
		} else {
			r = latest.HelmRelease{
				Name:        ch.name,
				ChartPath:   ch.path,
				Version:     ch.version,
				ValuesFiles: ch.valueFiles,
			}
		}
		releases = append(releases, r)
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
	// Run helm template in each dir.
	// Parse manifests and then get image names.
	artifacts := []string{}
	for _, ch := range h.charts {
		args := []string{"template", ch.path}
		for _, v := range ch.valueFiles {
			args = append(args, "-f", v)
		}
		args = append(args, "--dry-run")
		cmd := exec.Command("helm", args...)
		mb, err := util.RunCmdOut(context.TODO(), cmd)
		if err != nil {
			log.Entry(context.TODO()).Warnf("could not initialize builder for helm chart %q.\nCommand %q encountered error: %s", ch.name, cmd, err)
			continue
		}
		images, err := kubernetes.ParseImagesFromKubernetesYamlBytes(bufio.NewReader(bytes.NewReader(mb)))
		if err != nil {
			log.Entry(context.TODO()).Warnf("could not initialize builder for helm chart %q.\nCould not parse %q output due to error: %s", ch.name, cmd, err)
		} else {
			artifacts = append(artifacts, images...)
		}
	}
	return artifacts
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

func getRemoteChart(parsed map[string]interface{}) []chart {
	var remotes []chart
	if deps, ok := parsed["dependencies"]; ok {
		list := deps.([]map[string]interface{})
		for _, r := range list {
			ch := chart{
				name:     r["name"].(string),
				isRemote: true,
				repo:     r["repository"].(string),
			}
			if v := getVersion(r); v != "" {
				ch.version = v
			}
			remotes = append(remotes)
		}
	}
	return remotes
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
