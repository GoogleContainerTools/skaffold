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
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildSpecFail(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latestV2.Artifact
	}{
		{
			description: "bazel",
			artifact: &latestV2.Artifact{
				ArtifactType: latestV2.ArtifactType{
					BazelArtifact: &latestV2.BazelArtifact{},
				},
			},
		},
		{
			description: "unknown",
			artifact:    &latestV2.Artifact{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder := NewBuilder(&mockBuilderContext{}, &latestV2.GoogleCloudBuild{})

			_, err := builder.buildSpec(context.Background(), test.artifact, "tag", platform.Matcher{}, "bucket", "object")

			t.CheckError(true, err)
		})
	}
}

type mockBuilderContext struct {
	runcontext.RunContext // Embedded to provide the default values.
	artifactStore         build.ArtifactStore
	sourceDepsResolver    func() graph.SourceDependenciesCache
}

func (c *mockBuilderContext) SourceDependenciesResolver() graph.SourceDependenciesCache {
	if c.sourceDepsResolver != nil {
		return c.sourceDepsResolver()
	}
	return nil
}

func (c *mockBuilderContext) ArtifactStore() build.ArtifactStore { return c.artifactStore }
