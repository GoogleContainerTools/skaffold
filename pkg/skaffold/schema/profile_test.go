/*
Copyright 2018 The Skaffold Authors

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

package schema

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestApplyProfiles(t *testing.T) {
	tests := []struct {
		description string
		config      *latest.SkaffoldPipeline
		profile     string
		expected    *latest.SkaffoldPipeline
		shouldErr   bool
	}{
		{
			description: "unknown profile",
			config:      config(),
			profile:     "profile",
			expected:    config(),
			shouldErr:   true,
		},
		{
			description: "build type",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Build: latest.BuildConfig{
						BuildType: latest.BuildType{
							GoogleCloudBuild: &latest.GoogleCloudBuild{
								ProjectID: "my-project",
							},
						},
					},
				}),
			),
			expected: config(
				withGoogleCloudBuild("my-project",
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "tag policy",
			profile:     "dev",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "dev",
					Build: latest.BuildConfig{
						TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withShaTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "artifacts",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{ImageName: "image"},
							{ImageName: "imageProd"},
						},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("image", ".", "Dockerfile"),
					withDockerArtifact("imageProd", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "deploy",
			profile:     "profile",
			config: config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withProfiles(latest.Profile{
					Name: "profile",
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							HelmDeploy: &latest.HelmDeploy{},
						},
					},
				}),
			),
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withHelmDeploy(),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := ApplyProfiles(test.config, []string{test.profile})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, test.config)
		})
	}
}
