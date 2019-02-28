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

package bazel

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetArtifact(t *testing.T) {
	tests := []struct {
		name      string
		initial   *latest.Artifact
		expected  *latest.Artifact
		shouldErr bool
	}{
		{
			name: "set target correctly",
			initial: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte("target: myTarget"),
				},
			},
			expected: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte("target: myTarget"),
				},
				ArtifactType: latest.ArtifactType{
					BazelArtifact: &latest.BazelArtifact{
						BuildTarget: "myTarget",
					},
				},
			},
		},
		{
			name: "set target and build args correctly",
			initial: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte(`target: myTarget
args:
  - arg1=arg1
  - arg2=arg2`)},
			},
			expected: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte(`target: myTarget
args:
  - arg1=arg1
  - arg2=arg2`)},
				ArtifactType: latest.ArtifactType{
					BazelArtifact: &latest.BazelArtifact{
						BuildTarget: "myTarget",
						BuildArgs: []string{
							"arg1=arg1",
							"arg2=arg2",
						},
					},
				},
			},
		},
		{
			name: "no target",
			initial: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte(`args:
  - arg=arg`)},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := setArtifact(test.initial)
			if test.shouldErr {
				testutil.CheckError(t, test.shouldErr, err)
				return
			}
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, test.initial)
		})
	}
}
