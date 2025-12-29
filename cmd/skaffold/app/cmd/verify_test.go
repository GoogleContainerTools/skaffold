/*
Copyright 2024 The Skaffold Authors

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

package cmd

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetVerifyImgs(t *testing.T) {
	tests := []struct {
		description string
		configs     []util.VersionedConfig
		expected    map[string]bool
	}{
		{
			description: "no verify config",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{},
			},
			expected: map[string]bool{},
		},
		{
			description: "single verify test case",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
						},
					},
				},
			},
			expected: map[string]bool{"image1": true},
		},
		{
			description: "multiple verify test cases",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
							{Container: latest.VerifyContainer{Image: "image2"}},
						},
					},
				},
			},
			expected: map[string]bool{"image1": true, "image2": true},
		},
		{
			description: "multiple configs",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
						},
					},
				},
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image2"}},
						},
					},
				},
			},
			expected: map[string]bool{"image1": true, "image2": true},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := getVerifyImgs(test.configs)
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestTargetArtifactsForVerify(t *testing.T) {
	tests := []struct {
		description string
		configs     []util.VersionedConfig
		expected    []*latest.Artifact
	}{
		{
			description: "no artifacts or verify config",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{},
			},
			expected: nil,
		},
		{
			description: "build artifact matches verify image",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image1"},
							},
						},
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
						},
					},
				},
			},
			expected: []*latest.Artifact{{ImageName: "image1"}},
		},
		{
			description: "build artifact not used in verify",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image1"},
								{ImageName: "image2"},
							},
						},
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
						},
					},
				},
			},
			expected: []*latest.Artifact{{ImageName: "image1"}},
		},
		{
			description: "verify image not in build artifacts (external image)",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image1"},
							},
						},
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "alpine:latest"}},
						},
					},
				},
			},
			expected: nil,
		},
		{
			description: "multiple configs with mixed artifacts",
			configs: []util.VersionedConfig{
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image1"},
								{ImageName: "unused"},
							},
						},
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image1"}},
						},
					},
				},
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "image2"},
							},
						},
						Verify: []*latest.VerifyTestCase{
							{Container: latest.VerifyContainer{Image: "image2"}},
						},
					},
				},
			},
			expected: []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := targetArtifactsForVerify(test.configs)
			t.CheckDeepEqual(test.expected, result)
		})
	}
}
