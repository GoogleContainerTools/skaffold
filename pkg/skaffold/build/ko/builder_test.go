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

package ko

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		description              string
		artifact                 latestV1.Artifact
		runMode                  config.RunMode
		wantDisableOptimizations bool
		wantLabels               []string
		wantPlatform             string
		wantWorkingDirectory     string
	}{
		{
			description: "all zero value",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
			},
		},
		{
			description: "base image",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						BaseImage: "gcr.io/distroless/base:nonroot",
					},
				},
			},
		},
		{
			description: "empty platforms",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Platforms: []string{},
					},
				},
			},
		},
		{
			description: "multiple platforms",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Platforms: []string{"linux/amd64", "linux/arm64"},
					},
				},
			},
			wantPlatform: "linux/amd64,linux/arm64",
		},
		{
			description: "workspace",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				Workspace: "my-app-subdirectory",
			},
			wantWorkingDirectory: "my-app-subdirectory",
		},
		{
			description: "source dir",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Dir: "my-go-mod-is-here",
					},
				},
			},
			wantWorkingDirectory: "my-go-mod-is-here",
		},
		{
			description: "workspace and source dir",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Dir: "my-go-mod-is-here",
					},
				},
				Workspace: "my-app-subdirectory",
			},
			wantWorkingDirectory: "my-app-subdirectory" + string(filepath.Separator) + "my-go-mod-is-here",
		},
		{
			description: "disable compiler optimizations for debug",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
			},
			runMode:                  config.RunModes.Debug,
			wantDisableOptimizations: true,
		},
		{
			description: "labels",
			artifact: latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Labels: map[string]string{
							"foo":  "bar",
							"frob": "baz",
						},
					},
				},
			},
			wantLabels: []string{"foo=bar", "frob=baz"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			bo := buildOptions(&test.artifact, test.runMode)
			t.CheckDeepEqual(test.artifact.KoArtifact.BaseImage, bo.BaseImage)
			if bo.ConcurrentBuilds < 1 {
				t.Errorf("ConcurrentBuilds must always be >= 1 for the ko builder")
			}
			t.CheckDeepEqual(test.wantPlatform, bo.Platform)
			t.CheckDeepEqual(version.UserAgentWithClient(), bo.UserAgent)
			t.CheckDeepEqual(test.wantWorkingDirectory, bo.WorkingDirectory)
			t.CheckDeepEqual(test.wantDisableOptimizations, bo.DisableOptimizations)
			t.CheckDeepEqual(test.wantLabels, bo.Labels,
				cmpopts.SortSlices(func(x, y string) bool { return x < y }),
				cmpopts.EquateEmpty())
		})
	}
}
