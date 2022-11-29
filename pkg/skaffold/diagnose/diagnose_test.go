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

package diagnose

import (
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestSizeOfDockerContext(t *testing.T) {
	tests := []struct {
		description        string
		artifactName       string
		DockerfileContents string
		files              map[string]string
		expected           int64
		shouldErr          bool
	}{
		{
			description:        "test size",
			artifactName:       "empty",
			DockerfileContents: "From Scratch",
			expected:           2048,
		},
		{
			description:        "test size for a image with file",
			artifactName:       "image",
			DockerfileContents: "From Scratch \n Copy foo /",
			files:              map[string]string{"foo": "foo"},
			expected:           3072,
		},
		{
			description:        "incorrect docker file",
			artifactName:       "error-artifact",
			DockerfileContents: "From Scratch \n Copy doesNotExists /",
			shouldErr:          true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("Dockerfile", test.DockerfileContents).
				WriteFiles(test.files)

			dummyArtifact := &latest.Artifact{
				Workspace: tmpDir.Root(),
				ImageName: test.artifactName,
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}

			actual, err := sizeOfDockerContext(context.TODO(), dummyArtifact, nil)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestCheckArtifacts(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("Dockerfile", "FROM busybox")

		err := CheckArtifacts(context.Background(), &mockConfig{
			artifacts: []*latest.Artifact{{
				Workspace: tmpDir.Root(),
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}},
		}, io.Discard)

		t.CheckNoError(err)
	})
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	artifacts             []*latest.Artifact
}

func (c *mockConfig) PipelineForImage() latest.Pipeline {
	var pipeline latest.Pipeline
	pipeline.Build.Artifacts = c.artifacts
	return pipeline
}

func (c *mockConfig) GetPipelines() []latest.Pipeline {
	var pipelines []latest.Pipeline
	pipelines = append(pipelines, c.PipelineForImage())
	return pipelines
}

func (c *mockConfig) Artifacts() []*latest.Artifact {
	return c.artifacts
}
