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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetArtifact(t *testing.T) {
	tests := []struct {
		name     string
		initial  *latest.Artifact
		expected *latest.Artifact
	}{
		{
			name: "no contents passed in",
			initial: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{},
			},
			expected: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{},
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			},
		},
		{
			name: "set dockerfile path",
			initial: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte("dockerfile: path/to/Dockerfile"),
				},
			},
			expected: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte("dockerfile: path/to/Dockerfile"),
				},
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "path/to/Dockerfile",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := setArtifact(test.initial)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, test.initial)
		})
	}
}
