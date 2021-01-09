/*
Copyright 2021 The Skaffold Authors

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

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
)

const (
	minimalConfig = ``

	simpleConfig = `
build:
  tagPolicy:
    gitCommit: {}
  artifacts:
  - image: example
deploy:
  kubectl: {}
`
	// This config has two tag policies set.
	invalidConfig = `
build:
  tagPolicy:
    sha256: {}
    gitCommit: {}
  artifacts:
  - image: example
deploy:
  name: example
`

	completeConfig = `
apiVersion: skaffold/v2beta11
kind: Config
metadata:
  name: %s
%s
build:
  artifacts:
  - image: image%d
profiles:
- name: pf0
  build:
    artifacts:
    - image: pf0image%d
- name: pf1
  build:
    artifacts:
    - image: pf1image%d
`
)

func createCfg(name string, imageName string, workspace string, requires []latest.ConfigDependency) *latest.SkaffoldConfig {
	return &latest.SkaffoldConfig{
		APIVersion:   "skaffold/v2beta11",
		Kind:         "Config",
		Dependencies: requires,
		Metadata:     latest.Metadata{Name: name},
		Pipeline: latest.Pipeline{Build: latest.BuildConfig{
			Artifacts: []*latest.Artifact{{ImageName: imageName, ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{DockerfilePath: "Dockerfile"}}, Workspace: workspace}}, TagPolicy: latest.TagPolicy{
				GitTagger: &latest.GitTagger{}}, BuildType: latest.BuildType{
				LocalBuild: &latest.LocalBuild{Concurrency: concurrency()},
			}}, Deploy: latest.DeployConfig{Logs: latest.LogsConfig{Prefix: "container"}}},
	}
}

func concurrency() *int {
	c := 1
	return &c
}

type document struct {
	path    string
	configs []mockCfg
}

type mockCfg struct {
	name           string
	requiresStanza string
}

func TestGetAllConfigs(t *testing.T) {
	tests := []struct {
		description  string
		documents    []document
		configFilter []string
		profiles     []string
		err          error
		expected     []*latest.SkaffoldConfig
	}{
		{
			description: "no dependencies",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg0", requiresStanza: ""}}}},
			expected:    []*latest.SkaffoldConfig{createCfg("cfg0", "image0", ".", nil)},
		},
		{
			description:  "no dependencies, config flag",
			documents:    []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg0", requiresStanza: ""}, {name: "cfg1", requiresStanza: ""}}}},
			expected:     []*latest.SkaffoldConfig{createCfg("cfg1", "image1", ".", nil)},
			configFilter: []string{"cfg1"},
		},
		{
			description:  "no dependencies, config flag, profiles flag",
			documents:    []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg0", requiresStanza: ""}, {name: "cfg1", requiresStanza: ""}}}},
			expected:     []*latest.SkaffoldConfig{createCfg("cfg1", "pf0image1", ".", nil)},
			configFilter: []string{"cfg1"},
			profiles:     []string{"pf0"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			for _, d := range test.documents {
				var cfgs []string
				for i, c := range d.configs {
					s := fmt.Sprintf(completeConfig, c.name, c.requiresStanza, i, i, i)
					cfgs = append(cfgs, s)
				}
				tmpDir.Write(d.path, strings.Join(cfgs, "\n---\n"))
			}

			cfgs, err := getAllConfigs(config.SkaffoldOptions{
				ConfigurationFile: tmpDir.Path(test.documents[0].path),
				Configuration:     test.configFilter,
				Profiles:          test.profiles,
			})

			t.CheckDeepEqual(test.err, err, cmp.Comparer(errorsComparer))
			t.CheckErrorAndDeepEqual(test.err != nil, err, test.expected, cfgs)
		})
	}
}

func errorsComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Error() == b.Error()
}
