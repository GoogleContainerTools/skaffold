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
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
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

func createCfg(name string, imageName string, workspace string, requires []latestV2.ConfigDependency) *latestV2.SkaffoldConfig {
	return &latestV2.SkaffoldConfig{
		APIVersion:   latestV2.Version,
		Kind:         "Config",
		Dependencies: requires,
		Metadata:     latestV2.Metadata{Name: name},
		Pipeline: latestV2.Pipeline{Build: latestV2.BuildConfig{
			Artifacts: []*latestV2.Artifact{{ImageName: imageName, ArtifactType: latestV2.ArtifactType{
				DockerArtifact: &latestV2.DockerArtifact{DockerfilePath: "Dockerfile"}}, Workspace: workspace}}, TagPolicy: latestV2.TagPolicy{
				GitTagger: &latestV2.GitTagger{}}, BuildType: latestV2.BuildType{
				LocalBuild: &latestV2.LocalBuild{Concurrency: concurrency()},
			}}, Deploy: latestV2.DeployConfig{Logs: latestV2.LogsConfig{Prefix: "container"}}},
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
		description              string
		documents                []document
		configFilter             []string
		profiles                 []string
		makePathsAbsolute        *bool
		errCode                  proto.StatusCode
		applyProfilesRecursively bool
		expected                 func(base string) []*latestV2.SkaffoldConfig
	}{
		{
			description: "makePathsAbsolute unspecified; no dependencies",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg00", "image00", ".", nil)}
			},
		},
		{
			description: "makePathsAbsolute unspecified; no dependencies, config flag",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "image01", ".", nil)}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description: "makePathsAbsolute unspecified; no dependencies, config flag, profiles flag",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "pf0image01", ".", nil)}
			},
			configFilter: []string{"cfg01"},
			profiles:     []string{"pf0"},
		},
		{
			description: "makePathsAbsolute unspecified; branch dependencies",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; cascading dependencies",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; self dependency",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg01", "image01", ".", nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Names: []string{"cfg01"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; dependencies in same file",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1"}}),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; looped dependencies",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), []latestV2.ConfigDependency{{Path: base, Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; dependencies with profile in root, not in dependent",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; dependencies with profile in dependent activated by profile in root",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; dependencies with auto-activated profile in dependent (no `activatedBy` clause)",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf1image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0"}}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; named profile not found",
			profiles:    []string{"pf0", "pf2"},
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
			errCode: proto.StatusCode_CONFIG_PROFILES_NOT_FOUND_ERR,
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
				}
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
			description: "makePathsAbsolute unspecified; duplicate config names across multiple configs",
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
			description: "makePathsAbsolute unspecified; duplicate config names in main config",
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00"}, {name: "cfg00"}}},
			},
			errCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
		},
		{
			description: "makePathsAbsolute unspecified; remote dependencies",
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1"}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; no dependencies",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg00", "image00", ".", nil)}
			},
		},
		{
			description:       "makePathsAbsolute false; no dependencies, config flag",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "image01", ".", nil)}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description:       "makePathsAbsolute false; no dependencies, config flag, profiles flag",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "pf0image01", ".", nil)}
			},
			configFilter: []string{"cfg01"},
			profiles:     []string{"pf0"},
		},
		{
			description:       "makePathsAbsolute false; branch dependencies",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg10", "image10", ".", nil),
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; cascading dependencies",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; self dependency",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg01", "image01", ".", nil),
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Names: []string{"cfg01"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; dependencies in same file",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg11", "image11", ".", nil),
					createCfg("cfg10", "image10", ".", []latestV2.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1"}}),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; looped dependencies",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", ".", []latestV2.ConfigDependency{{Path: "../", Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; dependencies with profile in root, not in dependent",
			makePathsAbsolute: util.BoolPtr(false),
			profiles:          []string{"pf0"},
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; dependencies with profile in dependent activated by profile in root",
			makePathsAbsolute: util.BoolPtr(false),
			profiles:          []string{"pf0"},
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; dependencies with auto-activated profile in dependent (no `activatedBy` clause)",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf1image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0"}}}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; named profile not found",
			makePathsAbsolute: util.BoolPtr(false),
			profiles:          []string{"pf0", "pf2"},
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
			errCode: proto.StatusCode_CONFIG_PROFILES_NOT_FOUND_ERR,
		},
		{
			description:       "cascading dependencies with config flag",
			makePathsAbsolute: util.BoolPtr(false),
			configFilter:      []string{"cfg11"},
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg11", "image11", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
				}
			},
		},
		{
			description:       "named config not found",
			makePathsAbsolute: util.BoolPtr(false),
			configFilter:      []string{"cfg3"},
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
			description:       "makePathsAbsolute false; duplicate config names across multiple configs",
			makePathsAbsolute: util.BoolPtr(false),
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
			description:       "makePathsAbsolute false; duplicate config names in main config",
			makePathsAbsolute: util.BoolPtr(false),
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00"}, {name: "cfg00"}}},
			},
			errCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
		},
		{
			description:       "makePathsAbsolute false; remote dependencies",
			makePathsAbsolute: util.BoolPtr(false),
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
			expected: func(string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", ".", []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latestV2.ConfigDependency{{Path: "doc1"}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; no dependencies",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg00", "image00", base, nil)}
			},
		},
		{
			description:       "makePathsAbsolute true; no dependencies, config flag",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "image01", base, nil)}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description:       "makePathsAbsolute true; no dependencies, config flag, profiles flag",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{createCfg("cfg01", "pf0image01", base, nil)}
			},
			configFilter: []string{"cfg01"},
			profiles:     []string{"pf0"},
		},
		{
			description:       "makePathsAbsolute true; branch dependencies",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}, {Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg01", "image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; cascading dependencies",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; self dependency",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg01", "image01", base, nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Names: []string{"cfg01"}}, {Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; dependencies in same file",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1")}}),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; looped dependencies",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), []latestV2.ConfigDependency{{Path: base, Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
					createCfg("cfg01", "image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; dependencies with profile in root, not in dependent",
			makePathsAbsolute: util.BoolPtr(true),
			profiles:          []string{"pf0"},
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; dependencies with profile in dependent activated by profile in root",
			makePathsAbsolute: util.BoolPtr(true),
			profiles:          []string{"pf0"},
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg01", "pf0image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; dependencies with auto-activated profile in dependent (no `activatedBy` clause)",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf1image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}, ActiveProfiles: []latestV2.ProfileDependency{{Name: "pf0"}}}}),
					createCfg("cfg01", "image01", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; named profile not found",
			makePathsAbsolute: util.BoolPtr(true),
			profiles:          []string{"pf0", "pf2"},
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
			errCode: proto.StatusCode_CONFIG_PROFILES_NOT_FOUND_ERR,
		},
		{
			description:       "makePathsAbsolute true; cascading dependencies with config flag",
			makePathsAbsolute: util.BoolPtr(true),
			configFilter:      []string{"cfg11"},
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; named config not found",
			makePathsAbsolute: util.BoolPtr(true),
			configFilter:      []string{"cfg3"},
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
			description:       "makePathsAbsolute true; duplicate config names across multiple configs",
			makePathsAbsolute: util.BoolPtr(true),
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
			description:       "makePathsAbsolute true; duplicate config names in main config",
			makePathsAbsolute: util.BoolPtr(true),
			documents: []document{
				{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00"}, {name: "cfg00"}}},
			},
			errCode: proto.StatusCode_CONFIG_DUPLICATE_NAMES_SAME_FILE_ERR,
		},
		{
			description:       "makePathsAbsolute true; remote dependencies",
			makePathsAbsolute: util.BoolPtr(true),
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{GitRepo: &latestV2.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1")}}),
					createCfg("cfg01", "image01", base, nil),
				}
			},
		},
		{
			description:              "makePathsAbsolute unspecified; recursively applied profiles",
			profiles:                 []string{"pf0"},
			applyProfilesRecursively: true,
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description:              "makePathsAbsolute false; recursively applied profiles",
			makePathsAbsolute:        util.BoolPtr(false),
			profiles:                 []string{"pf0"},
			applyProfilesRecursively: true,
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latestV2.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latestV2.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", ".", nil),
				}
			},
		},
		{
			description:              "makePathsAbsolute true; recursively applied profiles",
			makePathsAbsolute:        util.BoolPtr(true),
			profiles:                 []string{"pf0"},
			applyProfilesRecursively: true,
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
			expected: func(base string) []*latestV2.SkaffoldConfig {
				return []*latestV2.SkaffoldConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", base, []latestV2.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
					createCfg("cfg01", "pf0image01", base, nil),
				}
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
					s := fmt.Sprintf(template, latestV2.Version, c.name, c.requiresStanza, id, id, id)
					cfgs = append(cfgs, s)
				}
				tmpDir.Write(d.path, strings.Join(cfgs, "\n---\n"))
			}
			tmpDir.Chdir()
			var expected []*latestV2.SkaffoldConfig
			if test.expected != nil {
				wd, _ := util.RealWorkDir()
				expected = test.expected(wd)
			}
			t.Override(&git.SyncRepo, func(g latestV2.GitInfo, _ config.SkaffoldOptions) (string, error) { return g.Repo, nil })
			cfgs, err := GetAllConfigs(config.SkaffoldOptions{
				Command:             "dev",
				ConfigurationFile:   test.documents[0].path,
				ConfigurationFilter: test.configFilter,
				Profiles:            test.profiles,
				PropagateProfiles:   test.applyProfilesRecursively,
				MakePathsAbsolute:   test.makePathsAbsolute,
			})
			if test.errCode == proto.StatusCode_OK {
				t.CheckDeepEqual(expected, cfgs)
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
