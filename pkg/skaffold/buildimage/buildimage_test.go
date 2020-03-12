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

package buildimage

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestArtifact(t *testing.T) {
	tests := []struct {
		description      string
		options          Options
		expectedArtifact *latest.Artifact
	}{
		{
			description: "docker artifact",
			options:     Options{Name: "my-image", Workspace: ".", Type: "docker"},
			expectedArtifact: &latest.Artifact{
				ImageName: "my-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
		},
		{
			description: "docker artifact with target",
			options:     Options{Name: "my-image", Workspace: ".", Type: "docker", Target: "my-target"},
			expectedArtifact: &latest.Artifact{
				ImageName: "my-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						Target: "my-target",
					},
				},
			},
		},
		{
			description: "buildpacks artifact",
			options:     Options{Name: "other-image", Workspace: ".", Type: "buildpacks"},
			expectedArtifact: &latest.Artifact{
				ImageName: "other-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Builder: defaultBuildpacksBuilder,
					},
				},
			},
		},
		{
			description: "jib artifact",
			options:     Options{Name: "other-image", Workspace: ".", Type: "jib"},
			expectedArtifact: &latest.Artifact{
				ImageName: "other-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					JibArtifact: &latest.JibArtifact{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			a, err := artifact(test.options)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedArtifact, a)
		})
	}
}

func TestArtifactAutoDetect(t *testing.T) {
	tests := []struct {
		description      string
		options          Options
		files            map[string]string
		expectedArtifact *latest.Artifact
	}{
		{
			description: "docker artifact",
			options:     Options{Name: "my-image", Workspace: "."},
			files: map[string]string{
				"Dockerfile": "FROM scratch",
			},
			expectedArtifact: &latest.Artifact{
				ImageName: "my-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				},
			},
		},
		{
			description: "docker artifact with target",
			options:     Options{Name: "my-image", Workspace: ".", Target: "my-target"},
			files: map[string]string{
				"Dockerfile": "FROM scratch",
			},
			expectedArtifact: &latest.Artifact{
				ImageName: "my-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						Target: "my-target",
					},
				},
			},
		},
		{
			description: "buildpacks artifact",
			options:     Options{Name: "my-image", Workspace: "."},
			files: map[string]string{
				"go.mod": "something",
			},
			expectedArtifact: &latest.Artifact{
				ImageName: "my-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Builder: defaultBuildpacksBuilder,
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Chdir().WriteFiles(test.files)

			a, err := artifact(test.options)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedArtifact, a)
		})
	}
}
