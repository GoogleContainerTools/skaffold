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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetAbsFilePaths(t *testing.T) {
	tests := []struct {
		description string
		config      *latest.SkaffoldConfig
		base        string
		expected    *latest.SkaffoldConfig
	}{
		{
			description: "relative path",
			config: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{ImageName: "foo1", Workspace: "./foo"},
							{ImageName: "foo2", Workspace: "/a/foo"},
						},
					},
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy:     &latest.KptDeploy{Dir: "."},
							KubectlDeploy: &latest.KubectlDeploy{Manifests: []string{"foo/*", "/a/foo/*"}},
							HelmDeploy: &latest.HelmDeploy{Releases: []latest.HelmRelease{
								{ChartPath: "../charts", ValuesFiles: []string{"./values1.yaml", "./values2.yaml"}, SetFiles: map[string]string{"envFile": "./values3.yaml", "configFile": "./values4.yaml", "anotherFile": "/c/values5.yaml"}},
								{RemoteChart: "foo/bar", ValuesFiles: []string{"./values1.yaml", "./values2.yaml"}, SetFiles: map[string]string{"envFile": "./values3.yaml", "configFile": "./values4.yaml", "anotherFile": "/c/values5.yaml"}},
							}},
						},
					},
				},
			},
			base: "/a/b",
			expected: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{ImageName: "foo1", Workspace: "/a/b/foo"},
							{ImageName: "foo2", Workspace: "/a/foo"},
						},
					},
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy:     &latest.KptDeploy{Dir: "/a/b"},
							KubectlDeploy: &latest.KubectlDeploy{Manifests: []string{"/a/b/foo/*", "/a/foo/*"}},
							HelmDeploy: &latest.HelmDeploy{Releases: []latest.HelmRelease{
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
