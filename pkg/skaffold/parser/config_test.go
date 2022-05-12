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

// TODO(yuwenma 2021-07-27), multi-module not supported in v2 yet.

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/git"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemaUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/pkg/errors"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser/configlocations"
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
				LocalBuild: &latest.LocalBuild{Concurrency: util.IntPtr(1)},
			}},
			Deploy: latest.DeployConfig{Logs: latest.LogsConfig{Prefix: "container"}, DeployType: latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}}}},
	}
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
		expected                 func(base string) []schemaUtil.VersionedConfig
	}{
		{
			description: "makePathsAbsolute unspecified; no dependencies",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg00", "image00", ".", nil),
				}
			},
		},
		{
			description: "makePathsAbsolute unspecified; no dependencies, config flag",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", ".", nil),
				}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description: "makePathsAbsolute unspecified; no dependencies, config flag, profiles flag",
			documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "pf0image01", ".", nil),
				}
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", ".", nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Names: []string{"cfg01"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), []latest.ConfigDependency{{Path: base, Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf1image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0"}}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; no dependencies",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg00", "image00", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute false; no dependencies, config flag",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", ".", nil),
				}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description:       "makePathsAbsolute false; no dependencies, config flag, profiles flag",
			makePathsAbsolute: util.BoolPtr(false),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "pf0image01", ".", nil),
				}
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg10", "image10", ".", nil),
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", ".", nil),
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Names: []string{"cfg01"}}, {Path: "doc2", Names: []string{"cfg21"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg11", "image11", ".", nil),
					createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", ".", []latest.ConfigDependency{{Path: "../", Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf1image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0"}}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg11", "image11", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
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
			expected: func(string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", ".", nil),
					createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", ".", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}),
					createCfg("cfg01", "image01", ".", nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; no dependencies",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg00", "image00", base, nil),
				}
			},
		},
		{
			description:       "makePathsAbsolute true; no dependencies, config flag",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", base, nil),
				}
			},
			configFilter: []string{"cfg01"},
		},
		{
			description:       "makePathsAbsolute true; no dependencies, config flag, profiles flag",
			makePathsAbsolute: util.BoolPtr(true),
			documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "pf0image01", base, nil),
				}
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}, {Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg01", "image01", base, nil),
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Names: []string{"cfg01"}}, {Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Names: []string{"cfg11"}}}),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1")}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), []latest.ConfigDependency{{Path: base, Names: []string{"cfg00"}}}),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
					createCfg("cfg00", "pf0image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0", ActivatedBy: []string{"pf0"}}}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf1image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf1"}}}}),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}, ActiveProfiles: []latest.ProfileDependency{{Name: "pf0"}}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg11", "image11", filepath.Join(base, "doc1"), []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
					createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1")}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", ".", nil),
					createCfg("cfg10", "pf0image10", ".", []latest.ConfigDependency{{Path: "../doc2", Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", ".", []latest.ConfigDependency{{Path: "doc1", Names: []string{"cfg10"}}}),
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
			expected: func(base string) []schemaUtil.VersionedConfig {
				return []schemaUtil.VersionedConfig{
					createCfg("cfg21", "pf0image21", filepath.Join(base, "doc2"), nil),
					createCfg("cfg10", "pf0image10", filepath.Join(base, "doc1"), []latest.ConfigDependency{{Path: filepath.Join(base, "doc2"), Names: []string{"cfg21"}}}),
					createCfg("cfg00", "pf0image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1"), Names: []string{"cfg10"}}}),
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
					s := fmt.Sprintf(template, latest.Version, c.name, c.requiresStanza, id, id, id)
					cfgs = append(cfgs, s)
				}
				tmpDir.Write(d.path, strings.Join(cfgs, "\n---\n"))
			}
			tmpDir.Chdir()
			var expected []schemaUtil.VersionedConfig
			if test.expected != nil {
				wd, _ := util.RealWorkDir()
				expected = test.expected(wd)
			}
			t.Override(&git.SyncRepo, func(ctx context.Context, g latest.GitInfo, _ config.SkaffoldOptions) (string, error) {
				return g.Repo, nil
			})
			cfgs, err := GetAllConfigs(context.Background(), config.SkaffoldOptions{
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

var testSkaffoldYaml = `apiVersion: skaffold/v3alpha1
kind: Config
build:
  artifacts:
    - image: app-0
deploy:
  kubectl:
    manifests:
      - manifests-0
profiles:
  - name: profile-0
    build:
      artifacts:
        - image: app-0-profile
          context: app-0-profile
    deploy:
      kubectl:
        manifests:
          - manifests-0-profile
    patches:
      - op: replace
        path: /build/artifacts/0
        value:
          image: app-0-patch
      - op: add
        path: /deploy/kubectl/manifests/1
        value: 'manifests-1'
  - name: profile-1
    build:
      artifacts:
        - image: app-1-profile
          context: app-1-profile
    deploy:
      kubectl:
        manifests:
          - manifests-1-profile
`

func TestConfigLocationsParse(t *testing.T) {
	tests := []struct {
		description      string
		skaffoldYamlText string
		profiles         []string
		missingNodeCount int
		expected         [][]kyaml.Filter
	}{
		{
			description:      "find all expected yaml nodes for input skaffold.yaml file",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("kind")},
				{kyaml.Lookup("build")},
				{kyaml.Lookup("build", "artifacts")},
				{kyaml.Lookup("deploy")},
				{kyaml.Lookup("deploy", "kubectl")},
				{kyaml.Lookup("deploy", "kubectl", "manifests")},
			},
		},
		{
			description:      "verify profile nodes not in yaml nodes when there is no profile",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("build"), kyaml.Lookup("artifacts")},
			},
			missingNodeCount: 1,
		},
		{
			description:      "find all expected yaml nodes for input skaffold.yaml file and input profile",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-0"},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("kind")},
				{kyaml.Lookup("build")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("build"), kyaml.Lookup("artifacts")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}),
					kyaml.Lookup("patches"), kyaml.GetElementByIndex(0), kyaml.Lookup("value"), kyaml.Lookup("image")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}),
					kyaml.Lookup("patches"), kyaml.GetElementByIndex(1), kyaml.Lookup("value")},
				{kyaml.Lookup("deploy")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("deploy"), kyaml.Lookup("kubectl")},
				{kyaml.Lookup("profiles"), kyaml.MatchElementList([]string{"name"}, []string{"profile-0"}), kyaml.Lookup("deploy"), kyaml.Lookup("kubectl"), kyaml.Lookup("manifests")},
			},
		},
		{
			description:      "verify default nodes not in yaml nodes when there is an active profile overwriting the default node",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: [][]kyaml.Filter{
				{kyaml.Lookup("apiVersion")},
				{kyaml.Lookup("build", "artifacts")},
			},
			missingNodeCount: 1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			missingNodeCount := 0

			fp := t.TempFile("skaffoldyaml-", []byte(test.skaffoldYamlText))
			cfgs, err := GetConfigSet(context.TODO(), config.SkaffoldOptions{ConfigurationFile: fp, Profiles: test.profiles})
			if err != nil {
				t.Fatalf(err.Error())
			}
			root, err := kyaml.Parse(test.skaffoldYamlText)
			if err != nil {
				t.Fatalf(err.Error())
			}
			var seen bool
			for _, filters := range test.expected {
				seen = false
				expectedNode := root
				var err error
				for _, filter := range filters {
					expectedNode, err = expectedNode.Pipe(filter)
					if err != nil {
						t.Fatalf(err.Error())
					}
				}
				if expectedNode == nil {
					t.Errorf("test query led to nil node, should not be the case for kyaml filters: %v", filters)
				}
				for _, yamlInfos := range cfgs[0].YAMLInfos.GetYamlInfosCopy() {
					for _, v := range yamlInfos {
						if reflect.DeepEqual(expectedNode, v.RNode) {
							seen = true
						}
					}
				}
				if seen != true && test.missingNodeCount == 0 {
					str, _ := expectedNode.String()
					t.Errorf("unable to find expected yaml node text: %q in the generated yaml node map: %v", str, cfgs[0].YAMLInfos.GetYamlInfosCopy())
				}
				if seen != true && test.missingNodeCount > 0 {
					missingNodeCount++
					if missingNodeCount > test.missingNodeCount {
						t.Errorf("expected %d missing nodes in test, found %d missing nodes", test.missingNodeCount, missingNodeCount)
					}
				}
			}
		})
	}
}

func TestConfigLocationsLocate(t *testing.T) {
	tests := []struct {
		description      string
		skaffoldYamlText string
		profiles         []string
		expected         []configlocations.Location
	}{
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{},
			expected: []configlocations.Location{
				{
					StartLine:   5,
					StartColumn: 14,
					EndLine:     6,
					EndColumn:   0,
				},
			},
		},
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected with active profile with a patch",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-0"},
			expected: []configlocations.Location{
				{
					StartLine:   24,
					StartColumn: 18,
					EndLine:     25,
					EndColumn:   0,
				},
			},
		},
		{
			description:      "verify location for SkaffoldConfig.Build.Artifacts[0] is as expected with active profile with no patch",
			skaffoldYamlText: testSkaffoldYaml,
			profiles:         []string{"profile-1"},
			expected: []configlocations.Location{
				{
					StartLine:   31,
					StartColumn: 18,
					EndLine:     32,
					EndColumn:   0,
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fp := t.TempFile("skaffoldyaml-", []byte(test.skaffoldYamlText))
			cfgs, err := GetConfigSet(context.TODO(), config.SkaffoldOptions{ConfigurationFile: fp, Profiles: test.profiles})
			if err != nil {
				t.Fatalf(err.Error())
			}
			artifact0Location := cfgs.Locate(cfgs[0].SkaffoldConfig.Build.Artifacts[0])
			artifact0Location.SourceFile = ""
			t.CheckDeepEqual(&test.expected[0], artifact0Location)
		})
	}
}
