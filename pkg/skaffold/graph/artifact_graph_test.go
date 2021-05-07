/*
Copyright 2021 The Skaffold Authors

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

package graph

import (
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestToArtifactGraph_shouldGenerateArtifactGraph(t *testing.T) {
	testutil.Run(t, "generate artifacts graph", func(t *testutil.T) {
		artifacts := []*latestV1.Artifact{
			{
				ImageName: "1",
			},
			{
				ImageName: "2",
			},
		}

		var graph = ToArtifactGraph(artifacts)
		t.CheckNotNil(graph)
		t.CheckEmpty(graph.Dependencies(artifacts[0]))
		t.CheckEmpty(graph.Dependencies(artifacts[1]))
	})
}

func TestToArtifactGraph_shouldReturnDependencies(t *testing.T) {
	testutil.Run(t, "return artifact dependencies", func(t *testutil.T) {
		artifacts := []*latestV1.Artifact{
			{
				ImageName: "1",
				Dependencies: []*latestV1.ArtifactDependency{
					{
						ImageName: "randomImageName",
						Alias:     "alias",
					},
				},
			},
			{
				ImageName: "randomImageName",
			},
		}

		var graph = ToArtifactGraph(artifacts)
		t.CheckNotNil(graph)
		t.CheckEmpty(graph.Dependencies(artifacts[1]))
		t.CheckDeepEqual(graph.Dependencies(artifacts[0])[0], artifacts[1])
	})
}
