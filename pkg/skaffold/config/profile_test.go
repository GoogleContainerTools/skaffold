/*
Copyright 2018 Google LLC

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

package config

import (
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

func TestApplyProfiles(t *testing.T) {
	tests := []struct {
		description    string
		config         SkaffoldConfig
		profile        string
		expectedConfig SkaffoldConfig
		shouldErr      bool
	}{
		{
			description: "unknown profile",
			config:      SkaffoldConfig{},
			profile:     "profile",
			shouldErr:   true,
		},
		{
			description: "build type",
			profile:     "profile",
			config: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
					},
					BuildType: BuildType{
						LocalBuild: &LocalBuild{},
					},
				},
				Deploy: DeployConfig{},
				Profiles: []Profile{
					{
						Name: "profile",
						Build: BuildConfig{
							BuildType: BuildType{
								GoogleCloudBuild: &GoogleCloudBuild{},
							},
						},
					},
				},
			},
			expectedConfig: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
					},
					BuildType: BuildType{
						GoogleCloudBuild: &GoogleCloudBuild{},
					},
				},
				Deploy: DeployConfig{},
			},
		},
		{
			description: "tag policy",
			profile:     "dev",
			config: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
					},
					TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
				},
				Deploy: DeployConfig{},
				Profiles: []Profile{
					{
						Name: "dev",
						Build: BuildConfig{
							TagPolicy: TagPolicy{ShaTagger: &ShaTagger{}},
						},
					},
				},
			},
			expectedConfig: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
					},
					TagPolicy: TagPolicy{ShaTagger: &ShaTagger{}},
				},
				Deploy: DeployConfig{},
			},
		},
		{
			description: "artifacts",
			profile:     "profile",
			config: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
					},
					TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
				},
				Deploy: DeployConfig{},
				Profiles: []Profile{
					{
						Name: "profile",
						Build: BuildConfig{
							Artifacts: []*Artifact{
								{ImageName: "image"},
								{ImageName: "imageProd"},
							},
						},
					},
				},
			},
			expectedConfig: SkaffoldConfig{
				Build: BuildConfig{
					Artifacts: []*Artifact{
						{ImageName: "image"},
						{ImageName: "imageProd"},
					},
					TagPolicy: TagPolicy{GitTagger: &GitTagger{}},
				},
				Deploy: DeployConfig{},
			},
		},
		{
			description: "deploy",
			profile:     "profile",
			config: SkaffoldConfig{
				Build: BuildConfig{},
				Deploy: DeployConfig{
					DeployType: DeployType{
						KubectlDeploy: &KubectlDeploy{},
					},
				},
				Profiles: []Profile{
					{
						Name: "profile",
						Deploy: DeployConfig{
							DeployType: DeployType{
								HelmDeploy: &HelmDeploy{},
							},
						},
					},
				},
			},
			expectedConfig: SkaffoldConfig{
				Build: BuildConfig{},
				Deploy: DeployConfig{
					DeployType: DeployType{
						HelmDeploy: &HelmDeploy{},
					},
				},
			},
		}}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.config.ApplyProfiles([]string{test.profile})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedConfig, test.config)
		})
	}
}
