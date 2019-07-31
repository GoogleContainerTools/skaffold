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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildSpecFail(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.Artifact
	}{
		{
			description: "bazel",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BazelArtifact: &latest.BazelArtifact{},
				},
			},
		},
		{
			description: "unknown",
			artifact:    &latest.Artifact{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder := newBuilder(latest.GoogleCloudBuild{})

			_, err := builder.buildSpec(test.artifact, "tag", "bucket", "object")

			t.CheckError(true, err)
		})
	}
}

func newBuilder(gcb latest.GoogleCloudBuild) *Builder {
	return NewBuilder(&runcontext.RunContext{
		Opts: config.SkaffoldOptions{},
		Cfg: latest.Pipeline{
			Build: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &gcb,
				},
			},
		},
	})
}
