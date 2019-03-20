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

package plugin

import (
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/shared"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockBuilder struct {
	labels    map[string]string
	artifacts []build.Artifact
}

func (b *mockBuilder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return b.artifacts, nil
}

func (b *mockBuilder) Labels() map[string]string {
	return b.labels
}

func (b *mockBuilder) Init(opts *config.SkaffoldOptions, env *latest.ExecutionEnvironment) {}

func (b *mockBuilder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return nil, nil
}

func TestPluginBuilderLabels(t *testing.T) {
	tests := []struct {
		name     string
		builder  shared.PluginBuilder
		expected map[string]string
	}{
		{
			name: "check labels",
			builder: &Builder{
				Builders: map[string]shared.PluginBuilder{
					"mock-one": &mockBuilder{
						labels: map[string]string{"key-one": "value-one"},
					},
					"mock-two": &mockBuilder{
						labels: map[string]string{"key-two": "value-two"},
					},
				},
			},
			expected: map[string]string{
				"key-one": "value-one",
				"key-two": "value-two",
			},
		},
		{
			name: "check overlapping labels",
			builder: &Builder{
				Builders: map[string]shared.PluginBuilder{
					"mock-one": &mockBuilder{
						labels: map[string]string{"key-one": "value"},
					},
					"mock-two": &mockBuilder{
						labels: map[string]string{"key-one": "value"},
					},
				},
			},
			expected: map[string]string{
				"key-one":      "value",
				"key-one-rand": "value",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := randomID
			mockRandomID := func() string {
				return "rand"
			}
			randomID = mockRandomID
			defer func() {
				randomID = original
			}()

			actual := test.builder.Labels()
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
