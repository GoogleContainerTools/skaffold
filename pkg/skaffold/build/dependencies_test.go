/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTransitiveSourceDependenciesCache(t *testing.T) {
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
		t.Override(&getDependenciesFunc, func(_ context.Context, a *latest.Artifact, _ docker.Config, _ docker.ArtifactResolver) ([]string, error) {
			counts[a.ImageName]++
			return deps[a.ImageName], nil
		})

		r := NewTransitiveSourceDependenciesCache(nil, nil, g)
		d, err := r.ResolveForArtifact(context.Background(), g["img1"])
		t.CheckNoError(err)
		expectedDeps := []string{"file11", "file12", "file21", "file22", "file31", "file32", "file41", "file42", "file41", "file42"}
		t.CheckDeepEqual(expectedDeps, d)
		for _, v := range counts {
			t.CheckDeepEqual(v, 1)
		}
	})
}
