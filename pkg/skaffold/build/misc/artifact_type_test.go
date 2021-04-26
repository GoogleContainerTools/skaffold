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

package misc

import (
	"testing"

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestArtifactType(t *testing.T) {
	tests := []struct {
		description string
		want        string
		artifact    *latest_v1.Artifact
	}{
		{"docker", "docker", &latest_v1.Artifact{
			ArtifactType: latest_v1.ArtifactType{
				DockerArtifact: &latest_v1.DockerArtifact{},
			},
		}},
		{"kaniko", "kaniko", &latest_v1.Artifact{
			ArtifactType: latest_v1.ArtifactType{
				KanikoArtifact: &latest_v1.KanikoArtifact{},
			},
		}},
		{"docker+kaniko", "docker", &latest_v1.Artifact{
			ArtifactType: latest_v1.ArtifactType{
				DockerArtifact: &latest_v1.DockerArtifact{},
				KanikoArtifact: &latest_v1.KanikoArtifact{},
			},
		}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := ArtifactType(test.artifact)
			if got != test.want {
				t.Errorf("ArtifactType(%+v) = %q; want %q", test.artifact, got, test.want)
			}
		})
	}
}

func TestFormatArtifact(t *testing.T) {
	tests := []struct {
		description string
		want        string
		artifact    *latest_v1.Artifact
	}{
		{"docker", "docker: {}", &latest_v1.Artifact{
			ArtifactType: latest_v1.ArtifactType{
				DockerArtifact: &latest_v1.DockerArtifact{},
			},
		}},
		{"kaniko", "kaniko: {}", &latest_v1.Artifact{
			ArtifactType: latest_v1.ArtifactType{
				KanikoArtifact: &latest_v1.KanikoArtifact{},
			},
		}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got := FormatArtifact(test.artifact)
			if got != test.want {
				t.Errorf("FormatArtifact(%+v) = %q; want %q", test.artifact, got, test.want)
			}
		})
	}
}
