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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildSpecFail(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest_v1.Artifact
	}{
		{
			description: "bazel",
			artifact: &latest_v1.Artifact{
				ArtifactType: latest_v1.ArtifactType{
					BazelArtifact: &latest_v1.BazelArtifact{},
				},
			},
		},
		{
			description: "unknown",
			artifact:    &latest_v1.Artifact{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder := NewBuilder(&mockBuilderContext{}, &latest_v1.GoogleCloudBuild{})

			_, err := builder.buildSpec(test.artifact, "tag", "bucket", "object")

			t.CheckError(true, err)
		})
	}
}

type mockBuilderContext struct {
	runcontext.RunContext // Embedded to provide the default values.
	artifactStore         build.ArtifactStore
	sourceDepsResolver    func() graph.TransitiveSourceDependenciesCache
}

func (c *mockBuilderContext) SourceDependenciesResolver() graph.TransitiveSourceDependenciesCache {
	if c.sourceDepsResolver != nil {
		return c.sourceDepsResolver()
	}
	return nil
}

func (c *mockBuilderContext) ArtifactStore() build.ArtifactStore { return c.artifactStore }
