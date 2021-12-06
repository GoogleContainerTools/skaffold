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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSourceDependenciesCache(t *testing.T) {
	testutil.Run(t, "TestTransitiveSourceDependenciesCache", func(t *testutil.T) {
		g := map[string]*latestV2.Artifact{
			"img1": {ImageName: "img1", Dependencies: []*latestV2.ArtifactDependency{{ImageName: "img2"}}},
			"img2": {ImageName: "img2", Dependencies: []*latestV2.ArtifactDependency{{ImageName: "img3"}, {ImageName: "img4"}}},
			"img3": {ImageName: "img3", Dependencies: []*latestV2.ArtifactDependency{{ImageName: "img4"}}},
			"img4": {ImageName: "img4"},
		}
		deps := map[string][]string{
			"img1": {"file11", "file12"},
			"img2": {"file21", "file22"},
			"img3": {"file31", "file32"},
			"img4": {"file41", "file42"},
		}
		counts := map[string]int{"img1": 0, "img2": 0, "img3": 0, "img4": 0}
		t.Override(&getDependenciesFunc, func(_ context.Context, a *latestV2.Artifact, _ docker.Config, _ docker.ArtifactResolver) ([]string, error) {
			counts[a.ImageName]++
			return deps[a.ImageName], nil
		})

		r := NewSourceDependenciesCache(nil, nil, g)
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
		"dir1/baz.java",
		"dir2/frob.go",
	)
	tests := []struct {
		description            string
		artifact               *latestV1.Artifact
		dockerConfig           docker.Config
		dockerArtifactResolver docker.ArtifactResolver
		expectedPaths          []string
	}{
		{
			description: "ko default dependencies",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				Workspace: tmpDir.Root(),
			},
			expectedPaths: []string{
				filepath.Join(tmpDir.Root(), "dir2/frob.go"),
				filepath.Join(tmpDir.Root(), "bar.go"),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			paths, err := sourceDependenciesForArtifact(context.Background(), test.artifact, test.dockerConfig, test.dockerArtifactResolver)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedPaths, paths,
				cmpopts.SortSlices(func(x, y string) bool { return x < y }))
		})
	}
}
