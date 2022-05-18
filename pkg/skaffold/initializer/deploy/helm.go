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
	"io/ioutil"
	"os"
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
	tempDir  = ioutil.TempDir
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
func (h helm) DeployConfig() (latest.DeployConfig, []latest.Profile) {
	releases := []latest.HelmRelease{}
	for _, ch := range h.charts {
		r := latest.HelmRelease{
			Name:        ch.name,
			ChartPath:   ch.path,
			Version:     ch.version,
			ValuesFiles: ch.valueFiles,
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
	// Run helm template in each top level dir.
	// Parse templated manifest files and then get image names.
	artifacts := []string{}
	td, err := tempDir(os.TempDir(), "skaffold_")
	if err != nil {
		log.Entry(context.TODO()).Fatalf("cannot create temporary directory. Encountered error: %s", err)
	}
	defer os.RemoveAll(td)
	for _, ch := range h.charts {
		args := []string{"template", ch.path}
		for _, v := range ch.valueFiles {
			args = append(args, "-f", v)
		}
		o, err := tempDir(td, ch.name)
		if err != nil {
			log.Entry(context.TODO()).Fatalf("cannot create temporary directory. Encountered error: %s", err)
		}
		args = append(args, "--output-dir", o)
		cmd := exec.Command("helm", args...)
		err = util.RunCmd(context.TODO(), cmd)
		if err != nil {
			log.Entry(context.TODO()).Warnf("could not initialize builders for helm chart %q.\nCommand %q encountered error: %s", ch.name, cmd, err)
			continue
		}
		// read all templates generated
		files := getAllFiles(o)
		for _, file := range files {
			images, err := kubernetes.ParseImagesFromKubernetesYaml(file)
			if err != nil {
				log.Entry(context.TODO()).Warnf("could not initialize builder for helm chart %q.\nCould not parse %q output due to error: %s", ch.name, cmd, err)
			} else {
				artifacts = append(artifacts, images...)
			}
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

func getAllFiles(o string) []string {
	var files []string
	err := filepath.Walk(o, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		log.Entry(context.TODO()).Fatalf("could not walk directory %q due to error: %s", o, err)
	}
	return files
}
