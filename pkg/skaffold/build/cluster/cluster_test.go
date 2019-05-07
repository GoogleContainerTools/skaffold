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

package cluster

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSplitArtifacts(t *testing.T) {

	kanikoOne := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			KanikoArtifact: &latest.KanikoArtifact{
				Image: "image",
			},
		},
	}

	kanikoTwo := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			KanikoArtifact: &latest.KanikoArtifact{
				Image: "image2",
			},
		},
	}

	dockerArtifact := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{
				DockerfilePath: "path/to/Dockerfile",
			},
		},
	}

	bazelArtifact := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			BazelArtifact: &latest.BazelArtifact{
				BuildTarget: "://target",
			},
		},
	}

	tests := []struct {
		description    string
		artifacts      []*latest.Artifact
		expectedKaniko []*latest.Artifact
		expectedOther  []*latest.Artifact
	}{
		{
			description: "all kaniko",
			artifacts: []*latest.Artifact{
				kanikoOne, kanikoTwo,
			},
			expectedKaniko: []*latest.Artifact{
				kanikoOne, kanikoTwo,
			},
		}, {
			description: "all other",
			artifacts: []*latest.Artifact{
				bazelArtifact, dockerArtifact,
			},
			expectedOther: []*latest.Artifact{
				bazelArtifact, dockerArtifact,
			},
		}, {
			description: "mixture of kaniko and other",
			artifacts: []*latest.Artifact{
				kanikoOne, kanikoTwo, bazelArtifact, dockerArtifact,
			},
			expectedKaniko: []*latest.Artifact{
				kanikoOne, kanikoTwo,
			},
			expectedOther: []*latest.Artifact{
				bazelArtifact, dockerArtifact,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actualKaniko, actualOther := splitArtifacts(test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedKaniko, actualKaniko)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedOther, actualOther)
		})
	}
}
