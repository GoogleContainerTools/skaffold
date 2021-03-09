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

package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/git"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	template = `
apiVersion: %s
kind: Config
metadata:
  name: %s
%s
build:
  artifacts:
  - image: image%s
profiles:
- name: pf0
  build:
    artifacts:
    - image: pf0image%s
- name: pf1
  build:
    artifacts:
    - image: pf1image%s
`
)

func createCfg(name string, imageName string, workspace string, requires []latest.ConfigDependency) *latest.SkaffoldConfig {
	return &latest.SkaffoldConfig{
		APIVersion:   latest.Version,
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
		errCode      proto.StatusCode
		expected     []*latest.SkaffoldConfig
	}{
		{
			description: "no dependencies",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected:    []*latest.SkaffoldConfig{createCfg("cfg00", "image00", ".", nil)},
		},
		{
			description:  "no dependencies, config flag",
			documents:    []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected:     []*latest.SkaffoldConfig{createCfg("cfg01", "image01", ".", nil)},
			configFilter: []string{"cfg01"},
		},
		{
			description:  "no dependencies, config flag, profiles flag",
			documents:    []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected:     []*latest.SkaffoldConfig{createCfg("cfg01", "pf0image01", ".", nil)},
			configFilter: []string{"cfg01"},
			profiles:     []string{"pf0"},
		},
		{
			description: "branch dependencies",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
  - path: doc2
    configs: [cfg21]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: ""}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg10", "image10", "doc1", nil),
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
				createCfg("cfg01", "image01", ".", nil),
			},
		},
		{
			description: "cascading dependencies",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg10", "image10", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
				createCfg("cfg01", "image01", ".", nil),
			},
		},
		{
			description: "self dependency",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - configs: [cfg01]
  - path: doc2
    configs: [cfg21]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: ""}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg01", "image01", ".", nil),
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Names: []string{"cfg01"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
			},
		},
		{
			description: "dependencies in same file",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
`}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - configs: [cfg11]
`}, {name: "cfg11", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg11", "image11", "doc1", nil),
				createCfg("cfg10", "image10", "doc1", []latest.ConfigDependency{{Names: []string{"cfg11"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
			},
		},
		{
			description: "looped dependencies",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: `
requires:
  - path: ../
    configs: [cfg00]
`}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "image21", "doc2", []latest.ConfigDependency{{Path: "../", Names: []string{"cfg00"}}}),
				createCfg("cfg10", "image10", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
				createCfg("cfg01", "image01", ".", nil),
			},
		},
		{
			description: "dependencies with profile in root, not in dependent",
			profiles:    []string{"pf0"},
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg10", "image10", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
				createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
				createCfg("cfg01", "pf0image01", ".", nil),
			},
		},
		{
			description: "dependencies with profile in dependent activated by profile in root",
			profiles:    []string{"pf0"},
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
    activeProfiles:
      - name: pf0
        activatedBy: [pf0]
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
    activeProfiles:
      - name: pf0
        activatedBy: [pf0]
`}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "pf0image21", "doc2", nil),
				createCfg("cfg10", "pf0image10", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
				createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
				createCfg("cfg01", "pf0image01", ".", nil),
			},
		},
		{
			description: "dependencies with auto-activated profile in dependent (no `activatedBy` clause)",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
    activeProfiles:
      - name: pf0
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
    activeProfiles:
      - name: pf1
`}, {name: "cfg11", requiresStanza: ""}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "pf1image21", "doc2", nil),
				createCfg("cfg10", "pf0image10", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf1"}}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0"}}}}),
				createCfg("cfg01", "image01", ".", nil),
			},
		},
		{
			description:  "cascading dependencies with config flag",
			configFilter: []string{"cfg11"},
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg11]
`}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg11", "image11", "doc1", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
			},
		},
		{
			description:  "named config not found",
			configFilter: []string{"cfg3"},
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg11]
`}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: `
requires:
  - path: ../doc2
    configs: [cfg21]
`}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			errCode: proto.StatusCode_CONFIG_BAD_FILTER_ERR,
		},
		{
			description: "duplicate config names across multiple configs",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg10]
`}, {name: "cfg01", requiresStanza: `
requires:
  - path: doc1
    configs: [cfg11]
`}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10"}, {name: "cfg11"}, {name: "cfg00"}}},
			},
			errCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_ACROSS_FILES_ERR,
		},
		{
			description: "duplicate config names in main config",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00"}, {name: "cfg00"}}},
			},
			errCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
		},
		{
			description: "remote dependencies",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: `
requires:
  - path: doc1
`}, {name: "cfg01", requiresStanza: ""}}},
				{path: "doc1/skaffold.yaml", configs: []mockCfg{{name: "cfg10", requiresStanza: `
requires:
  - git:
      repo: doc2
      path: skaffold.yaml
      ref: main
    configs: [cfg21]
`}, {name: "cfg11", requiresStanza: `
requires:
  - git:
      repo: doc2
      ref: main
    configs: [cfg21]
`}}},
				{path: "doc2/skaffold.yaml", configs: []mockCfg{{name: "cfg20", requiresStanza: ""}, {name: "cfg21", requiresStanza: ""}}},
			},
			expected: []*latest.SkaffoldConfig{
				createCfg("cfg21", "image21", "doc2", nil),
				createCfg("cfg10", "image10", "doc1", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
				createCfg("cfg11", "image11", "doc1", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
				createCfg("cfg01", "image01", ".", nil),
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			for i, d := range test.documents {
				var cfgs []string
				for j, c := range d.configs {
					id := fmt.Sprintf("%d%d", i, j)
					s := fmt.Sprintf(template, latest.Version, c.name, c.requiresStanza, id, id, id)
					cfgs = append(cfgs, s)
				}
				tmpDir.Write(d.path, strings.Join(cfgs, "\n---\n"))
			}
			tmpDir.Chdir()
			for _, c := range test.expected {
				dir := c.Build.Artifacts[0].Workspace
				// in this test setup artifact workspace also denotes the config directory and no dependent config is in the root directory.
				if dir == "." {
					continue
				}
				// only for dependent configs update the expected path values to absolute.
				wd, _ := util.RealWorkDir()
				c.Build.Artifacts[0].Workspace = filepath.Join(wd, dir)
				for i := range c.Dependencies {
					if c.Dependencies[i].Path == "" {
						continue
					}
					c.Dependencies[i].Path = filepath.Join(wd, dir, c.Dependencies[i].Path)
				}
			}
			t.Override(&git.SyncRepo, func(g latest.GitInfo, _ config.SkaffoldOptions) (string, error) { return g.Repo, nil })
			cfgs, err := GetAllConfigs(config.SkaffoldOptions{
				Command:             "dev",
				ConfigurationFile:   test.documents[0].path,
				ConfigurationFilter: test.configFilter,
				Profiles:            test.profiles,
			})
			if test.errCode == proto.StatusCode_OK {
				t.CheckDeepEqual(test.expected, cfgs)
			} else {
				var e sErrors.Error
				if errors.As(err, &e) {
					t.CheckDeepEqual(test.errCode, e.StatusCode())
				} else {
					t.Fail()
				}
			}
		})
	}
}
