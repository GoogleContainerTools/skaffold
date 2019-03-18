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

package jib

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetMavenArtifact(t *testing.T) {
	tests := []struct {
		name      string
		initial   *latest.Artifact
		expected  *latest.Artifact
		shouldErr bool
	}{
		{
			name: "no contents passed in",
			initial: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{},
			},
			expected: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{},
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{},
				},
			},
		},
		{
			name: "set module correctly",
			initial: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{Contents: []byte(`module: mod`)},
			},
			expected: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{Contents: []byte(`module: mod`)},
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{Module: "mod"},
				},
			},
		},
		{
			name: "set profile correctly",
			initial: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{Contents: []byte(`profile: prof`)},
			},
			expected: &latest.Artifact{
				ImageName:     "image",
				BuilderPlugin: &latest.BuilderPlugin{Contents: []byte(`profile: prof`)},
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{Profile: "prof"},
				},
			},
		},
		{
			name: "set flags correctly",
			initial: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte(`args: ['--arg1=value1', '--arg2=value2']`),
				},
			},
			expected: &latest.Artifact{
				ImageName: "image",
				BuilderPlugin: &latest.BuilderPlugin{
					Contents: []byte(`args: ['--arg1=value1', '--arg2=value2']`),
				},
				ArtifactType: latest.ArtifactType{
					JibMavenArtifact: &latest.JibMavenArtifact{
						Flags: []string{"--arg1=value1", "--arg2=value2"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := setMavenArtifact(test.initial)
			if test.shouldErr {
				testutil.CheckError(t, test.shouldErr, err)
				return
			}
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, test.initial)
		})
	}
}
