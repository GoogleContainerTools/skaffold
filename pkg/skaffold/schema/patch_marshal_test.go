/*
Copyright 2026 The Skaffold Authors

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

package schema

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestMarshalForPatching(t *testing.T) {
	tests := []struct {
		description string
		config      latest.SkaffoldConfig
		expected    string
	}{
		{
			description: "nil collections are omitted",
			config: latest.SkaffoldConfig{Pipeline: latest.Pipeline{Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{{ImageName: "example", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}}},
			}}},
			expected: `apiVersion: ""
kind: ""
build:
  artifacts:
    - image: example
      docker: {}
`,
		},
		{
			description: "empty map is kept",
			config: latest.SkaffoldConfig{Pipeline: latest.Pipeline{Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{{ImageName: "example", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{BuildArgs: map[string]*string{}}}}},
			}}},
			expected: `apiVersion: ""
kind: ""
build:
  artifacts:
    - image: example
      docker:
        buildArgs: {}
`,
		},
		{
			description: "empty slice inside an otherwise empty struct is kept",
			config: latest.SkaffoldConfig{Pipeline: latest.Pipeline{Deploy: latest.DeployConfig{DeployType: latest.DeployType{
				LegacyHelmDeploy: &latest.LegacyHelmDeploy{Releases: []latest.HelmRelease{}},
			}}}},
			expected: `apiVersion: ""
kind: ""
deploy:
  helm:
    releases: []
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out, err := marshalForPatching(test.config)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, string(out))
		})
	}
}
