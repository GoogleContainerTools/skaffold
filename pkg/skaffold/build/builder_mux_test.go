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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewBuilderMux(t *testing.T) {
	tests := []struct {
		description         string
		pipelines           []latestV1.Pipeline
		pipeBuilder         func(latestV1.Pipeline) (PipelineBuilder, error)
		shouldErr           bool
		expectedBuilders    []string
		expectedConcurrency int
	}{
		{
			description: "only local builder",
			pipelines: []latestV1.Pipeline{
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{LocalBuild: &latestV1.LocalBuild{Concurrency: util.IntPtr(1)}}}},
			},
			pipeBuilder:         newMockPipelineBuilder,
			expectedBuilders:    []string{"local"},
			expectedConcurrency: 1,
		},
		{
			description: "only cluster builder",
			pipelines: []latestV1.Pipeline{
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{Cluster: &latestV1.ClusterDetails{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"cluster"},
		},
		{
			description: "only gcb builder",
			pipelines: []latestV1.Pipeline{
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{GoogleCloudBuild: &latestV1.GoogleCloudBuild{}}}},
			},
			pipeBuilder:      newMockPipelineBuilder,
			expectedBuilders: []string{"gcb"},
		},
		{
			description: "min non-zero concurrency",
			pipelines: []latestV1.Pipeline{
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{LocalBuild: &latestV1.LocalBuild{Concurrency: util.IntPtr(0)}}}},
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{LocalBuild: &latestV1.LocalBuild{Concurrency: util.IntPtr(3)}}}},
				{Build: latestV1.BuildConfig{BuildType: latestV1.BuildType{Cluster: &latestV1.ClusterDetails{Concurrency: 2}}}},
			},
			pipeBuilder:         newMockPipelineBuilder,
			expectedBuilders:    []string{"local", "local", "cluster"},
			expectedConcurrency: 2,
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
			t.CheckDeepEqual(test.expectedConcurrency, b.concurrency)
		})
	}
}

func TestGetConcurrency(t *testing.T) {
	tests := []struct {
		description         string
		pbs                 []PipelineBuilder
		cliConcurrency      int
		expectedConcurrency int
	}{
		{
			description: "default concurrency - builder and cli concurrency unset.",
			pbs: []PipelineBuilder{
				&mockPipelineBuilder{concurrency: nil, builderType: "local"},
				&mockPipelineBuilder{concurrency: nil, builderType: "gcb"},
			},
			cliConcurrency:      -1,
			expectedConcurrency: 1,
		},
		{
			description: "builder concurrency set to less than cli concurrency",
			pbs: []PipelineBuilder{
				&mockPipelineBuilder{concurrency: util.IntPtr(1), builderType: "local"},
				&mockPipelineBuilder{concurrency: util.IntPtr(1), builderType: "local"},
				&mockPipelineBuilder{concurrency: nil, builderType: "gcb"},
			},
			cliConcurrency:      2,
			expectedConcurrency: 2,
		},
		{
			description: "builder concurrency set",
			pbs: []PipelineBuilder{
				&mockPipelineBuilder{concurrency: util.IntPtr(2), builderType: "local"},
				&mockPipelineBuilder{concurrency: util.IntPtr(2), builderType: "local"},
				// As per docs https://github.com/GoogleContainerTools/skaffold/blob/dbd18994955f5805e80c6354ed0fd424ec4d987b/pkg/skaffold/schema/v2beta26/config.go#L287
				// nil concurrency defaults to 1
				&mockPipelineBuilder{concurrency: nil, builderType: "local"},
			},
			cliConcurrency:      -1,
			expectedConcurrency: 1,
		},
		{
			description: "builder concurrency set to 0 and cli concurrency set to 1",
			pbs: []PipelineBuilder{
				// build all in parallel
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "local"},
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "gcb"},
			},
			cliConcurrency:      1,
			expectedConcurrency: 1,
		},
		{
			description: "builder concurrency set to 0 and cli concurrency unset",
			pbs: []PipelineBuilder{
				// build all in parallel
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "local"},
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "gcb"},
			},
			cliConcurrency:      -1,
			expectedConcurrency: 0,
		},
		{
			description: "builder concurrency set to default 0 for gcb",
			pbs: []PipelineBuilder{
				// build all in parallel
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "gcb"},
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "gcb"},
			},
			cliConcurrency:      -1,
			expectedConcurrency: 0,
		},
		{
			description: "min non-zero concurrency",
			pbs: []PipelineBuilder{
				&mockPipelineBuilder{concurrency: util.IntPtr(0), builderType: "local"},
				&mockPipelineBuilder{concurrency: util.IntPtr(3), builderType: "gcb"},
				&mockPipelineBuilder{concurrency: util.IntPtr(2), builderType: "gcb"},
			},
			cliConcurrency:      -1,
			expectedConcurrency: 2,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := getConcurrency(test.pbs, test.cliConcurrency)
			t.CheckDeepEqual(test.expectedConcurrency, actual)
		})
	}
}

type mockConfig struct {
	pipelines []latestV1.Pipeline
	optRepo   string
}

func (m *mockConfig) GetPipelines() []latestV1.Pipeline { return m.pipelines }
func (m *mockConfig) GlobalConfig() string              { return "" }
func (m *mockConfig) DefaultRepo() *string {
	if m.optRepo != "" {
		return &m.optRepo
	}
	return nil
}
func (m *mockConfig) MultiLevelRepo() *bool { return nil }
func (m *mockConfig) BuildConcurrency() int { return -1 }

type mockPipelineBuilder struct {
	concurrency *int
	builderType string
}

func (m *mockPipelineBuilder) PreBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Build(ctx context.Context, out io.Writer, artifact *latestV1.Artifact) ArtifactBuilder {
	return nil
}

func (m *mockPipelineBuilder) PostBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Concurrency() *int { return m.concurrency }

func (m *mockPipelineBuilder) Prune(context.Context, io.Writer) error { return nil }

func (m *mockPipelineBuilder) PushImages() bool { return false }

func (m *mockPipelineBuilder) SupportedPlatforms() platform.Matcher { return platform.All }

func newMockPipelineBuilder(p latestV1.Pipeline) (PipelineBuilder, error) {
	switch {
	case p.Build.BuildType.LocalBuild != nil:
		return &mockPipelineBuilder{builderType: "local", concurrency: p.Build.LocalBuild.Concurrency}, nil
	case p.Build.BuildType.Cluster != nil:
		return &mockPipelineBuilder{builderType: "cluster", concurrency: util.IntPtr(p.Build.Cluster.Concurrency)}, nil
	case p.Build.BuildType.GoogleCloudBuild != nil:
		return &mockPipelineBuilder{builderType: "gcb", concurrency: util.IntPtr(p.Build.GoogleCloudBuild.Concurrency)}, nil
	default:
		return nil, errors.New("invalid config")
	}
}
