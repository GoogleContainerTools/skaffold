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
	"errors"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewBuilderMux(t *testing.T) {
	tests := []struct {
		description      string
		pipelines        []latest.Pipeline
		pipeBuilder      func(latest.Pipeline) (PipelineBuilder, error)
		shouldErr        bool
		expectedBuilders []string
	}{
		{
			description: "only local builder",
			pipelines: []latest.Pipeline{
				{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"local"},
		},
		{
			description: "only cluster builder",
			pipelines: []latest.Pipeline{
				{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"cluster"},
		},
		{
			description: "only gcb builder",
			pipelines: []latest.Pipeline{
				{Build: latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"gcb"},
		},
		{
			description: "multiple builders",
			pipelines: []latest.Pipeline{
				{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
				{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"local", "cluster"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := &mockConfig{pipelines: test.pipelines}

			b, err := NewBuilderMux(cfg, nil, test.pipeBuilder)
			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				return
			}
			t.CheckTrue(len(b.builders) == len(test.expectedBuilders))
			for i := range b.builders {
				t.CheckDeepEqual(test.expectedBuilders[i], b.builders[i].(*mockPipelineBuilder).builderType)
			}
		})
	}
}

type mockConfig struct {
	pipelines []latest.Pipeline
}

func (m *mockConfig) GetPipelines() []latest.Pipeline {
	return m.pipelines
}

type mockPipelineBuilder struct {
	concurrency int
	builderType string
}

func (m *mockPipelineBuilder) PreBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact) ArtifactBuilder {
	return nil
}

func (m *mockPipelineBuilder) PostBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Concurrency() int { return m.concurrency }

func (m *mockPipelineBuilder) Prune(context.Context, io.Writer) error { return nil }

func newMockPipelineBuilder(p latest.Pipeline) (PipelineBuilder, error) {
	switch {
	case p.Build.BuildType.LocalBuild != nil:
		return &mockPipelineBuilder{builderType: "local"}, nil
	case p.Build.BuildType.Cluster != nil:
		return &mockPipelineBuilder{builderType: "cluster"}, nil
	case p.Build.BuildType.GoogleCloudBuild != nil:
		return &mockPipelineBuilder{builderType: "gcb"}, nil
	default:
		return nil, errors.New("invalid config")
	}
}
