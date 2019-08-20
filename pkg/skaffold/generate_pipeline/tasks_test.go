/*
Copyright 2019 The Skaffold Authors

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

package generatepipeline

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateBuildTasks(t *testing.T) {
	var tests = []struct {
		description string
		configFiles []*ConfigFile
		shouldErr   bool
	}{
		{
			description: "successfully generate build tasks",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Profile: &latest.Profile{
						Pipeline: latest.Pipeline{
							Build: latest.BuildConfig{
								Artifacts: []*latest.Artifact{
									{
										ImageName: "testArtifact1",
									},
								},
							},
						},
					},
				},
				{
					Path: "test2",
					Profile: &latest.Profile{
						Pipeline: latest.Pipeline{
							Build: latest.BuildConfig{
								Artifacts: []*latest.Artifact{
									{
										ImageName: "testArtifact2",
									},
								},
							},
						},
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateBuildTasks(test.configFiles)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateBuildTask(t *testing.T) {
	var tests = []struct {
		description string
		buildConfig latest.BuildConfig
		shouldErr   bool
	}{
		{
			description: "successfully generate build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "testArtifact",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configFile := &ConfigFile{
				Path: "test",
				Profile: &latest.Profile{
					Pipeline: latest.Pipeline{
						Build: test.buildConfig,
					},
				},
			}
			_, err := generateBuildTask(configFile)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateDeployTasks(t *testing.T) {
	var tests = []struct {
		description string
		configFiles []*ConfigFile
		shouldErr   bool
	}{
		{
			description: "successfully generate deploy tasks",
			configFiles: []*ConfigFile{
				{
					Path: "test1",
					Config: &latest.SkaffoldConfig{
						Pipeline: latest.Pipeline{
							Deploy: latest.DeployConfig{
								DeployType: latest.DeployType{
									HelmDeploy: &latest.HelmDeploy{},
								},
							},
						},
					},
				},
				{
					Path: "test2",
					Config: &latest.SkaffoldConfig{
						Pipeline: latest.Pipeline{
							Deploy: latest.DeployConfig{
								DeployType: latest.DeployType{
									HelmDeploy: &latest.HelmDeploy{},
								},
							},
						},
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateDeployTasks(test.configFiles)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateDeployTask(t *testing.T) {
	var tests = []struct {
		description  string
		deployConfig latest.DeployConfig
		shouldErr    bool
	}{
		{
			description: "successfully generate deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy: &latest.HelmDeploy{},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy:      nil,
					KubectlDeploy:   nil,
					KustomizeDeploy: nil,
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configFile := &ConfigFile{
				Path: "test",
				Config: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Deploy: test.deployConfig,
					},
				},
			}

			_, err := generateDeployTask(configFile)
			t.CheckError(test.shouldErr, err)
		})
	}
}
