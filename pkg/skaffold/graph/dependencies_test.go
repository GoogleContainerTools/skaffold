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

package graph

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestSourceDependenciesCache(t *testing.T) {
	testutil.Run(t, "TestTransitiveSourceDependenciesCache", func(t *testutil.T) {
		g := map[string]*latest.Artifact{
			"img1": {ImageName: "img1", Dependencies: []*latest.ArtifactDependency{{ImageName: "img2"}}},
			"img2": {ImageName: "img2", Dependencies: []*latest.ArtifactDependency{{ImageName: "img3"}, {ImageName: "img4"}}},
			"img3": {ImageName: "img3", Dependencies: []*latest.ArtifactDependency{{ImageName: "img4"}}},
			"img4": {ImageName: "img4"},
		}
		deps := map[string][]string{
			"img1": {"file11", "file12"},
			"img2": {"file21", "file22"},
			"img3": {"file31", "file32"},
			"img4": {"file41", "file42"},
		}
		counts := map[string]int{"img1": 0, "img2": 0, "img3": 0, "img4": 0}
		t.Override(&getDependenciesFunc, func(_ context.Context, a *latest.Artifact, _ docker.Config, _ docker.ArtifactResolver, _ string) ([]string, error) {
			counts[a.ImageName]++
			return deps[a.ImageName], nil
		})

		r := NewSourceDependenciesCache(nil, docker.NewSimpleStubArtifactResolver(), g)
		d, err := r.TransitiveArtifactDependencies(context.Background(), g["img1"])
		t.CheckNoError(err)
		expectedDeps := []string{"file11", "file12", "file21", "file22", "file31", "file32", "file41", "file42", "file41", "file42"}
		t.CheckDeepEqual(expectedDeps, d)
		for _, v := range counts {
			t.CheckDeepEqual(v, 1)
		}
	})
}

func TestSourceDependenciesForArtifact(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch(
		"foo.java",
		"bar.go",
		"latest.go",
		"dir1/baz.java",
		"dir2/frob.go",
	)
	tests := []struct {
		description        string
		artifact           *latest.Artifact
		tag                string
		dockerConfig       docker.Config
		dockerBuildArgs    map[string]string
		dockerFileContents string
		expectedPaths      []string
	}{
		{
			description: "ko default dependencies",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
				Workspace: tmpDir.Root(),
			},
			tag: "gcr.io/distroless/base:latest",
			expectedPaths: []string{
				filepath.Join(tmpDir.Root(), "dir2/frob.go"),
				filepath.Join(tmpDir.Root(), "bar.go"),
				filepath.Join(tmpDir.Root(), "latest.go"),
			},
		},
		{
			description: "docker default dependencies",
			artifact: &latest.Artifact{
				ImageName: "img1",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
				Workspace:    tmpDir.Root(),
				Dependencies: []*latest.ArtifactDependency{{ImageName: "img2", Alias: "BASE"}},
			},
			dockerBuildArgs: map[string]string{
				"IMAGE_REPO": "{{.IMAGE_REPO}}",
				"IMAGE_NAME": "{{.IMAGE_NAME}}",
				"IMAGE_TAG":  "{{.IMAGE_TAG}}",
			},
			dockerConfig: docker.NewConfigStub(config.RunModes.Build, false),
			dockerFileContents: `ARG IMAGE_REPO
ARG IMAGE_NAME
ARG IMAGE_TAG
FROM $IMAGE_REPO/$IMAGE_NAME:$IMAGE_TAG
COPY bar.go .
COPY $IMAGE_TAG.go .
`,
			expectedPaths: []string{
				filepath.Join(tmpDir.Root(), "Dockerfile"),
				filepath.Join(tmpDir.Root(), "bar.go"),
				filepath.Join(tmpDir.Root(), "latest.go"),
			},
			tag: "gcr.io/distroless/base:latest",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.RetrieveImage, docker.NewFakeImageFetcher())
			d := docker.NewSimpleStubArtifactResolver()
			tmpDir.Write("Dockerfile", test.dockerFileContents)
			if test.dockerBuildArgs != nil {
				args := map[string]*string{}
				for k, v := range test.dockerBuildArgs {
					args[k] = &v
				}
				test.artifact.DockerArtifact.BuildArgs = args
			}
			paths, err := sourceDependenciesForArtifact(context.Background(), test.artifact, test.dockerConfig, d, test.tag)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedPaths, paths,
				cmpopts.SortSlices(func(x, y string) bool { return x < y }))
		})
	}
}
