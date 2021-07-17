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
	"io/ioutil"
	"testing"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateProfile(t *testing.T) {
	var tests = []struct {
		description     string
		skaffoldConfig  *latestV2.SkaffoldConfig
		expectedProfile *latestV2.Profile
		responses       []string
		namespace       string
		shouldErr       bool
	}{
		{
			description: "successful profile generation docker",
			skaffoldConfig: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV2.ArtifactType{
									DockerArtifact: &latestV2.DockerArtifact{},
								},
							},
						},
					},
				},
			},
			namespace: "",
			expectedProfile: &latestV2.Profile{
				Name: "oncluster",
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV2.ArtifactType{
									KanikoArtifact: &latestV2.KanikoArtifact{},
								},
							},
						},
						BuildType: latestV2.BuildType{
							Cluster: &latestV2.ClusterDetails{
								PullSecretName: "kaniko-secret",
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "successful profile generation jib",
			skaffoldConfig: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV2.ArtifactType{
									JibArtifact: &latestV2.JibArtifact{
										Project: "test-module",
									},
									DockerArtifact: nil,
								},
							},
						},
					},
				},
			},
			namespace: "",
			expectedProfile: &latestV2.Profile{
				Name: "oncluster",
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV2.ArtifactType{
									JibArtifact: &latestV2.JibArtifact{
										Project: "test-module",
									},
								},
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "kaniko artifact with namespace",
			skaffoldConfig: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV2.ArtifactType{
									DockerArtifact: &latestV2.DockerArtifact{},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			expectedProfile: &latestV2.Profile{
				Name: "oncluster",
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV2.ArtifactType{
									KanikoArtifact: &latestV2.KanikoArtifact{},
								},
							},
						},
						BuildType: latestV2.BuildType{
							Cluster: &latestV2.ClusterDetails{
								PullSecretName: "kaniko-secret",
								Namespace:      "test-ns",
							},
						},
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "failed profile generation",
			skaffoldConfig: &latestV2.SkaffoldConfig{
				Pipeline: latestV2.Pipeline{
					Build: latestV2.BuildConfig{
						Artifacts: []*latestV2.Artifact{},
					},
				},
			},
			expectedProfile: nil,
			shouldErr:       true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			profile, err := generateProfile(ioutil.Discard, test.namespace, test.skaffoldConfig)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedProfile, profile)
		})
	}
}
