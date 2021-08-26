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

package debugger

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigurationsAndImages(t *testing.T) {
	tests := []struct {
		name      string
		artifacts []graph.Artifact
	}{
		{
			name:      "no artifacts doesn't error",
			artifacts: []graph.Artifact{},
		},
		{
			name: "one artifact, one configuration",
			artifacts: []graph.Artifact{
				{ImageName: "foo", Tag: "foo:bar"},
			},
		},
		{
			name: "two artifacts, two configurations",
			artifacts: []graph.Artifact{
				{ImageName: "foo", Tag: "foo:bar"},
				{ImageName: "another", Tag: "another:image"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&TransformImage, func(_ context.Context, a graph.Artifact, _ *container.Config, _ map[string]bool, _ string) (map[string]types.ContainerDebugConfiguration, []*container.Config, error) {
				m := make(map[string]types.ContainerDebugConfiguration)
				m[a.ImageName] = types.ContainerDebugConfiguration{
					Artifact: a.ImageName,
				}
				return m, nil, nil
			})

			m := NewDebugManager(nil, "")

			for _, a := range test.artifacts {
				m.TransformImage(context.TODO(), a, &container.Config{Image: a.ImageName})
			}

			for _, a := range test.artifacts {
				if !findArtifactInImageList(m.images, a) {
					t.Errorf("unable to find artifact %+v in image list: %v", a, m.images)
				}

				if !validateConfiguration(m.configurations, a) {
					t.Errorf("unable to find configuration for artifact %+v in map: %+v", a, m.configurations)
				}
			}
		})
	}
}

func findArtifactInImageList(list []string, target graph.Artifact) bool {
	var found bool
	for _, item := range list {
		if item == target.ImageName {
			found = true
		}
	}
	return found
}

func validateConfiguration(config map[string]types.ContainerDebugConfiguration, a graph.Artifact) bool {
	var validated bool
	for i, c := range config {
		if i == a.ImageName {
			validated = c.Artifact == a.ImageName
		}
	}
	return validated
}
