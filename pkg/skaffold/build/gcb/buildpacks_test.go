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

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildpackBuildSpec(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.BuildpackArtifact
		expected    cloudbuild.Build
		shouldErr   bool
	}{
		{
			description: "default run image",
			artifact: &latest.BuildpackArtifact{
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
			artifact: &latest.BuildpackArtifact{
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
			artifact: &latest.BuildpackArtifact{
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
			description: "invalid env",
			artifact: &latest.BuildpackArtifact{
				Builder: "builder",
				Env:     []string{"FOO={{INVALID}}"},
			},
			shouldErr: true,
		},
		{
			description: "buildpacks list",
			artifact: &latest.BuildpackArtifact{
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
			artifact: &latest.BuildpackArtifact{
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
			artifact: &latest.BuildpackArtifact{
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

			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: test.artifact,
				},
			}

			builder := newBuilder(latest.GoogleCloudBuild{
				PackImage: "pack/image",
			})
			buildSpec, err := builder.buildSpec(artifact, "img", "bucket", "object")
			t.CheckError(test.shouldErr, err)

			if !test.shouldErr {
				t.CheckDeepEqual(test.expected.Steps, buildSpec.Steps)
				t.CheckDeepEqual(test.expected.Images, buildSpec.Images)
			}
		})
	}
}
