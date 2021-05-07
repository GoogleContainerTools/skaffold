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

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateProfile(t *testing.T) {
	var tests = []struct {
		description     string
		skaffoldConfig  *latestV1.SkaffoldConfig
		expectedProfile *latestV1.Profile
		responses       []string
		namespace       string
		shouldErr       bool
	}{
		{
			description: "successful profile generation docker",
			skaffoldConfig: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV1.ArtifactType{
									DockerArtifact: &latestV1.DockerArtifact{},
								},
							},
						},
					},
				},
			},
			namespace: "",
			expectedProfile: &latestV1.Profile{
				Name: "oncluster",
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV1.ArtifactType{
									KanikoArtifact: &latestV1.KanikoArtifact{},
								},
							},
						},
						BuildType: latestV1.BuildType{
							Cluster: &latestV1.ClusterDetails{
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
			skaffoldConfig: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV1.ArtifactType{
									JibArtifact: &latestV1.JibArtifact{
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
			expectedProfile: &latestV1.Profile{
				Name: "oncluster",
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV1.ArtifactType{
									JibArtifact: &latestV1.JibArtifact{
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
			skaffoldConfig: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test",
								ArtifactType: latestV1.ArtifactType{
									DockerArtifact: &latestV1.DockerArtifact{},
								},
							},
						},
					},
				},
			},
			namespace: "test-ns",
			expectedProfile: &latestV1.Profile{
				Name: "oncluster",
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{
							{
								ImageName: "test-pipeline",
								ArtifactType: latestV1.ArtifactType{
									KanikoArtifact: &latestV1.KanikoArtifact{},
								},
							},
						},
						BuildType: latestV1.BuildType{
							Cluster: &latestV1.ClusterDetails{
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
			skaffoldConfig: &latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Build: latestV1.BuildConfig{
						Artifacts: []*latestV1.Artifact{},
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
