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

package gcb

import (
	"testing"

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildpackBuildSpec(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.BuildpackArtifact
		expected    cloudbuild.Build
	}{
		{
			description: "default run image",
			artifact: &latest.BuildpackArtifact{
				Builder: "builder",
			},
			expected: cloudbuild.Build{
				Options: &cloudbuild.BuildOptions{
					Volumes: []*cloudbuild.Volume{{Name: "layers", Path: "/layers"}},
				},
				Steps: []*cloudbuild.BuildStep{
					{
						Name: "busybox",
						Args: []string{"sh", "-c", "chown -R 1000:1000 /workspace /layers $$HOME"},
					},
					{
						Name:       "builder",
						Entrypoint: "/lifecycle/detector",
					},
					{
						Name:       "builder",
						Entrypoint: "/lifecycle/analyzer",
						Args:       []string{"img"},
					},
					{
						Name:       "builder",
						Entrypoint: "/lifecycle/builder",
					},
					{
						Name:       "builder",
						Entrypoint: "/lifecycle/exporter",
						Args:       []string{"img"},
					},
				},
			},
		},
		{
			description: "run image",
			artifact: &latest.BuildpackArtifact{
				Builder:  "otherbuilder",
				RunImage: "run/image",
			},
			expected: cloudbuild.Build{
				Options: &cloudbuild.BuildOptions{
					Volumes: []*cloudbuild.Volume{{Name: "layers", Path: "/layers"}},
				},
				Steps: []*cloudbuild.BuildStep{
					{
						Name: "busybox",
						Args: []string{"sh", "-c", "chown -R 1000:1000 /workspace /layers $$HOME"},
					},
					{
						Name:       "otherbuilder",
						Entrypoint: "/lifecycle/detector",
					},
					{
						Name:       "otherbuilder",
						Entrypoint: "/lifecycle/analyzer",
						Args:       []string{"img"},
					},
					{
						Name:       "otherbuilder",
						Entrypoint: "/lifecycle/builder",
					},
					{
						Name: "docker/docker",
						Args: []string{"pull", "run/image"},
					},
					{
						Name:       "otherbuilder",
						Entrypoint: "/lifecycle/exporter",
						Args:       []string{"-image", "run/image", "img"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: test.artifact,
				},
			}

			builder := newBuilder(latest.GoogleCloudBuild{
				DockerImage: "docker/docker",
			})
			buildSpec, err := builder.buildSpec(artifact, "img", "bucket", "object")
			t.CheckNoError(err)

			t.CheckDeepEqual(test.expected.Steps, buildSpec.Steps)
			t.CheckDeepEqual(test.expected.Options.Volumes, buildSpec.Options.Volumes)
			t.CheckDeepEqual(0, len(buildSpec.Images))
		})
	}
}
