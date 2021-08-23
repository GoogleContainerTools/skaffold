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

package inspect

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestModifyGcbBuildEnv(t *testing.T) {
	tests := []struct {
		description     string
		profile         string
		modules         []string
		buildEnvOpts    inspect.BuildEnvOptions
		expectedConfigs []string
		errCode         proto.StatusCode
		strict          bool
	}{
		{
			description:  "modify default pipeline; strict true",
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project2"},
			strict:       true,
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project2
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
- name: p2
  build:
    cluster: {}
`, ``,
			},
		},
		{
			description:  "modify profile pipeline; strict true",
			strict:       true,
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project2", Profile: "p1"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project1
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project2
- name: p2
  build:
    cluster: {}
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project2
`,
			},
		},
		{
			description:  "add to non-existing profile; strict true",
			strict:       true,
			buildEnvOpts: inspect.BuildEnvOptions{MachineType: "machine2", Concurrency: 2, Profile: "p3"},
			errCode:      proto.StatusCode_INSPECT_PROFILE_NOT_FOUND_ERR,
		},
		{
			description:  "add to profile with wrong build env type; strict true",
			strict:       true,
			buildEnvOpts: inspect.BuildEnvOptions{MachineType: "machine2", Concurrency: 2, Profile: "p2"},
			errCode:      proto.StatusCode_INSPECT_BUILD_ENV_INCORRECT_TYPE_ERR,
		},
		{
			description:  "modify default pipeline; strict false",
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project2"},
			strict:       false,
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project2
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
- name: p2
  build:
    cluster: {}
`, ``,
			},
		},
		{
			description:  "modify profile pipeline; strict false",
			strict:       false,
			buildEnvOpts: inspect.BuildEnvOptions{ProjectID: "project2", Profile: "p1"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project1
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project2
- name: p2
  build:
    cluster: {}
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project2
`,
			},
		},
		{
			description:  "add to non-existing profile; strict false",
			strict:       false,
			buildEnvOpts: inspect.BuildEnvOptions{MachineType: "machine2", Concurrency: 2, Profile: "p3"},
			errCode:      proto.StatusCode_INSPECT_PROFILE_NOT_FOUND_ERR,
		},
		{
			description:  "add to profile with wrong build env type; strict false",
			strict:       false,
			buildEnvOpts: inspect.BuildEnvOptions{MachineType: "machine2", Concurrency: 2, Profile: "p2"},
			expectedConfigs: []string{
				`apiVersion: ""
kind: ""
metadata:
  name: cfg1
requires:
- path: path/to/cfg2
build:
  googleCloudBuild:
    projectId: project1
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
- name: p2
  build:
    cluster: {}
`, `apiVersion: ""
kind: ""
metadata:
  name: cfg2
build:
  local: {}
profiles:
- name: p1
  build:
    googleCloudBuild:
      projectId: project1
`,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata:     v1.Metadata{Name: "cfg1"},
					Dependencies: []v1.ConfigDependency{{Path: pathToCfg2}},
					Pipeline:     v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{GoogleCloudBuild: &v1.GoogleCloudBuild{ProjectID: "project1"}}}},
					Profiles: []v1.Profile{
						{Name: "p1", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{GoogleCloudBuild: &v1.GoogleCloudBuild{ProjectID: "project1"}}}}},
						{Name: "p2", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{Cluster: &v1.ClusterDetails{}}}}},
					}}, SourceFile: pathToCfg1, IsRootConfig: true, SourceIndex: 0},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata: v1.Metadata{Name: "cfg2"},
					Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{LocalBuild: &v1.LocalBuild{}}}},
					Profiles: []v1.Profile{
						{Name: "p1", Pipeline: v1.Pipeline{Build: v1.BuildConfig{BuildType: v1.BuildType{GoogleCloudBuild: &v1.GoogleCloudBuild{ProjectID: "project1"}}}}},
					}}, SourceFile: pathToCfg2, SourceIndex: 0},
			}
			t.Override(&inspect.GetConfigSet, func(ctx context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				if len(opts.ConfigurationFilter) == 0 || util.StrSliceContains(opts.ConfigurationFilter, "cfg1") {
					return configSet, nil
				}
				if util.StrSliceContains(opts.ConfigurationFilter, "cfg2") {
					return parser.SkaffoldConfigSet{configSet[0]}, nil
				}
				return nil, nil
			})
			t.Override(&inspect.ReadFileFunc, func(filename string) ([]byte, error) {
				if filename == pathToCfg1 {
					return yaml.MarshalWithSeparator([]*v1.SkaffoldConfig{configSet[0].SkaffoldConfig})
				} else if filename == pathToCfg2 {
					return yaml.MarshalWithSeparator([]*v1.SkaffoldConfig{configSet[1].SkaffoldConfig})
				}
				t.FailNow()
				return nil, nil
			})
			var actualCfg1, actualCfg2 string
			t.Override(&inspect.WriteFileFunc, func(filename string, data []byte) error {
				switch filename {
				case pathToCfg1:
					actualCfg1 = string(data)
				case pathToCfg2:
					actualCfg2 = string(data)
				default:
					t.FailNow()
				}
				return nil
			})

			var buf bytes.Buffer
			err := ModifyGcbBuildEnv(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.modules, BuildEnvOptions: test.buildEnvOpts, Strict: test.strict})
			t.CheckNoError(err)
			if test.errCode == proto.StatusCode_OK {
				t.CheckDeepEqual(test.expectedConfigs[0], actualCfg1)
				t.CheckDeepEqual(test.expectedConfigs[1], actualCfg2)
			} else {
				t.CheckContains(test.errCode.String(), buf.String())
			}
		})
	}
}
