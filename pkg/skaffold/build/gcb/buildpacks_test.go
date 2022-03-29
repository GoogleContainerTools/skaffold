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

	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildpackBuildSpec(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latestV2.BuildpackArtifact
		expected    cloudbuild.Build
		shouldErr   bool
	}{
		{
			description: "default run image",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "builder",
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "builder"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "env variables",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "builder",
				Env:               []string{"KEY=VALUE", "FOO={{.BAR}}"},
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "builder", "--env", "KEY=VALUE", "--env", "FOO=bar"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "run image",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "otherbuilder",
				RunImage:          "run/image",
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "otherbuilder", "--run-image", "run/image"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "custom build image",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "img2",
				RunImage:          "run/image",
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "img2:tag", "--run-image", "run/image"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "custom run image",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "otherbuilder",
				RunImage:          "img3",
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "otherbuilder", "--run-image", "img3:tag"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "custom build and run image",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "img2",
				RunImage:          "img3",
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "img2:tag", "--run-image", "img3:tag"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "invalid env",
			artifact: &latestV2.BuildpackArtifact{
				Builder: "builder",
				Env:     []string{"FOO={{INVALID}}"},
			},
			shouldErr: true,
		},
		{
			description: "buildpacks list",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "builder",
				Buildpacks:        []string{"buildpack1", "buildpack2"},
				ProjectDescriptor: "project.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "builder", "--buildpack", "buildpack1", "--buildpack", "buildpack2"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "trusted builder",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "builder",
				ProjectDescriptor: "project.toml",
				TrustBuilder:      true,
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "builder", "--trust-builder"},
				}},
				Images: []string{"img"},
			},
		},
		{
			description: "project descriptor",
			artifact: &latestV2.BuildpackArtifact{
				Builder:           "builder",
				ProjectDescriptor: "non-default.toml",
			},
			expected: cloudbuild.Build{
				Steps: []*cloudbuild.BuildStep{{
					Name: "pack/image",
					Args: []string{"pack", "build", "img", "--builder", "builder", "--descriptor", "non-default.toml"},
				}},
				Images: []string{"img"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return []string{"BAR=bar"} })

			artifact := &latestV2.Artifact{
				ImageName: "img",
				ArtifactType: latestV2.ArtifactType{
					BuildpackArtifact: test.artifact,
				},
				Dependencies: []*latestV2.ArtifactDependency{{ImageName: "img2", Alias: "img2"}, {ImageName: "img3", Alias: "img3"}},
			}
			store := mockArtifactStore{
				"img2": "img2:tag",
				"img3": "img3:tag",
			}
			builder := NewBuilder(&mockBuilderContext{artifactStore: store}, &latestV2.GoogleCloudBuild{
				PackImage: "pack/image",
			})
			buildSpec, err := builder.buildSpec(context.Background(), artifact, "img", platform.Matcher{}, "bucket", "object")
			t.CheckError(test.shouldErr, err)

			if !test.shouldErr {
				t.CheckDeepEqual(test.expected.Steps, buildSpec.Steps)
				t.CheckDeepEqual(test.expected.Images, buildSpec.Images)
			}
		})
	}
}
