// +build !windows

/*
Copyright 2020 The Skaffold Authors

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

package tags

import (
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetAbsFilePaths(t *testing.T) {
	tests := []struct {
		description string
		config      *latestV1.SkaffoldConfig
		base        string
		expected    *latestV1.SkaffoldConfig
	}{
		{
			description: "relative path",
			config: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{ImageName: "foo1", Workspace: "./foo"},
							{ImageName: "foo2", Workspace: "/a/foo"},
						},
					},
					Deploy: latestV1.DeployConfig{
						DeployType: latestV1.DeployType{
							KptDeploy:     &latestV1.KptDeploy{Dir: "."},
							KubectlDeploy: &latestV1.KubectlDeploy{Manifests: []string{"foo/*", "/a/foo/*"}},
							HelmDeploy: &latestV1.HelmDeploy{Releases: []latestV1.HelmRelease{
								{ChartPath: "../charts", ValuesFiles: []string{"./values1.yaml", "./values2.yaml"}, SetFiles: map[string]string{"envFile": "./values3.yaml", "configFile": "./values4.yaml", "anotherFile": "/c/values5.yaml"}},
								{RemoteChart: "foo/bar", ValuesFiles: []string{"./values1.yaml", "./values2.yaml"}, SetFiles: map[string]string{"envFile": "./values3.yaml", "configFile": "./values4.yaml", "anotherFile": "/c/values5.yaml"}},
							}},
						},
					},
				},
			},
			base: "/a/b",
			expected: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{ImageName: "foo1", Workspace: "/a/b/foo"},
							{ImageName: "foo2", Workspace: "/a/foo"},
						},
					},
					Deploy: latestV1.DeployConfig{
						DeployType: latestV1.DeployType{
							KptDeploy:     &latestV1.KptDeploy{Dir: "/a/b"},
							KubectlDeploy: &latestV1.KubectlDeploy{Manifests: []string{"/a/b/foo/*", "/a/foo/*"}},
							HelmDeploy: &latestV1.HelmDeploy{Releases: []latestV1.HelmRelease{
								{ChartPath: "/a/charts", ValuesFiles: []string{"/a/b/values1.yaml", "/a/b/values2.yaml"}, SetFiles: map[string]string{"envFile": "/a/b/values3.yaml", "configFile": "/a/b/values4.yaml", "anotherFile": "/c/values5.yaml"}},
								{RemoteChart: "foo/bar", ValuesFiles: []string{"/a/b/values1.yaml", "/a/b/values2.yaml"}, SetFiles: map[string]string{"envFile": "/a/b/values3.yaml", "configFile": "/a/b/values4.yaml", "anotherFile": "/c/values5.yaml"}},
							}},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := MakeFilePathsAbsolute(test.config, test.base)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, test.config)
		})
	}
}
