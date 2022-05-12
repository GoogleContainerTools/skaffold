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
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemaUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type document struct {
	path    string
	configs []mockCfg
}

type mockCfg struct {
	name           string
	requiresStanza string
}

type testCase struct {
	description              string
	documents                []document
	configFilter             []string
	profiles                 []string
	makePathsAbsolute        *bool
	errCode                  proto.StatusCode
	applyProfilesRecursively bool
	expected                 func(string) []schemaUtil.VersionedConfig
}

func createCfg(name string, imageName string, workspace string, requires []latest.ConfigDependency) *latest.SkaffoldConfig {
	yamls := strings.Join([]string{workspace, "k8s/*.yaml"}, "/")
	if workspace == "." {
		yamls = "k8s/*.yaml"
	}
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
			Manifests: latest.RenderConfig{Generate: latest.Generate{RawK8s: []string{yamls}}},
			Deploy:    latest.DeployConfig{Logs: latest.LogsConfig{Prefix: "container"}, DeployType: latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}}}},
	}
}

var tcs = []testCase{
	{
		description: "makePathsAbsolute unspecified; no dependencies",
		documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
		expected: func(base string) []schemaUtil.VersionedConfig {
			return []schemaUtil.VersionedConfig{
				createCfg("cfg00", "image00", ".", nil),
			}
		},
	},
	{
		description: "makePathsAbsolute unspecified; no dependencies, config flag",
		documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
		expected: func(base string) []schemaUtil.VersionedConfig {
			return []schemaUtil.VersionedConfig{
				createCfg("cfg01", "image01", ".", nil),
			}
		},
		configFilter: []string{"cfg01"},
	},
	{
		description: "makePathsAbsolute unspecified; no dependencies, config flag, profiles flag",
		documents:   []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
		expected: func(base string) []schemaUtil.VersionedConfig {
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
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}, ""),
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
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}, ""),
				createCfg("cfg01", "image01", ".", nil),
			}
		},
	},
	{
		description:       "makePathsAbsolute false; no dependencies",
		makePathsAbsolute: util.BoolPtr(false),
		documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}}}},
		expected: func(base string) []schemaUtil.VersionedConfig {
			return []schemaUtil.VersionedConfig{
				createCfg("cfg00", "image00", ".", nil),
			}
		},
	},
	{
		description:       "makePathsAbsolute false; no dependencies, config flag",
		makePathsAbsolute: util.BoolPtr(false),
		documents:         []document{{path: "skaffold.yaml", configs: []mockCfg{{name: "cfg00", requiresStanza: ""}, {name: "cfg01", requiresStanza: ""}}}},
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
			return []schemaUtil.VersionedConfig{
				createCfg("cfg11", "image11", ".", nil),
				createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{Names: []string{"cfg11"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}, ""),
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
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
		expected: func(base string) []schemaUtil.VersionedConfig {
			return []schemaUtil.VersionedConfig{
				createCfg("cfg21", "image21", ".", nil),
				createCfg("cfg10", "image10", ".", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Path: "skaffold.yaml", Ref: "main"}, Names: []string{"cfg21"}}}),
				createCfg("cfg11", "image11", ".", []latest.ConfigDependency{{GitRepo: &latest.GitInfo{Repo: "doc2", Ref: "main"}, Names: []string{"cfg21"}}}),
				createCfg("cfg00", "image00", ".", []latest.ConfigDependency{{Path: "doc1"}}, ""),
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
				createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1")}}, ""),
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
				createCfg("cfg00", "image00", base, []latest.ConfigDependency{{Path: filepath.Join(base, "doc1")}}, ""),
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
