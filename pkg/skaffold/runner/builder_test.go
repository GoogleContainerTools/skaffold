/*
Copyright 2023 The Skaffold Authors

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

package runner

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPipelineBuilderWithHooks(t *testing.T) {
	type testcase struct {
		name             string
		hooks            latest.BuildHooks
		expected         []byte
		wantPreHooksErr  bool
		wantPostHooksErr bool
	}

	testcases := []testcase{
		{
			name:             "no hooks to execute",
			hooks:            latest.BuildHooks{},
			expected:         nil,
			wantPreHooksErr:  false,
			wantPostHooksErr: false,
		},
		{
			name: "execute pre-hooks in order",
			hooks: latest.BuildHooks{
				PreHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "echo hello world 1",
						},
					},
					{
						Command: []string{
							"sh", "-c", "echo hello world 2",
						},
					},
				},
				PostHooks: nil,
			},
			expected: []byte(
				"Starting pre-build hooks...\n" +
					"hello world 1\n" +
					"hello world 2\n" +
					"Completed pre-build hooks\n",
			),
		},
		{
			name: "execute post-hooks in order",
			hooks: latest.BuildHooks{
				PreHooks: nil,
				PostHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "echo hello world 1",
						},
					},
					{
						Command: []string{
							"sh", "-c", "echo hello world 2",
						},
					},
				},
			},
			expected: []byte(
				"Starting post-build hooks...\n" +
					"hello world 1\n" +
					"hello world 2\n" +
					"Completed post-build hooks\n",
			),
		},
		{
			name: "execute pre-hooks before post-hooks in order",
			hooks: latest.BuildHooks{
				PreHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "echo hello world 1",
						},
					},
				},
				PostHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "echo hello world 2",
						},
					},
				},
			},
			expected: []byte(
				"Starting pre-build hooks...\n" +
					"hello world 1\n" +
					"Completed pre-build hooks\n" +
					"Starting post-build hooks...\n" +
					"hello world 2\n" +
					"Completed post-build hooks\n",
			),
		},
		{
			name: "executing pre-hooks returns an error if one of the commands fail",
			hooks: latest.BuildHooks{
				PreHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "exit 1",
						},
					},
				},
				PostHooks: nil,
			},
			wantPreHooksErr: true,
		},
		{
			name: "executing post-hooks returns an error if one of the commands fail",
			hooks: latest.BuildHooks{
				PreHooks: nil,
				PostHooks: []latest.HostHook{
					{
						Command: []string{
							"sh", "-c", "exit 1",
						},
					},
				},
			},
			wantPostHooksErr: true,
		},
	}

	pipelineBuilder := &mockPipelineBuilder{}

	for _, tc := range testcases {
		tc := tc

		testutil.Run(t, tc.name, func(t *testutil.T) {
			ctx := context.Background()
			buf := new(bytes.Buffer)

			pb := withPipelineBuildHooks(pipelineBuilder, tc.hooks)

			err := pb.PreBuild(ctx, buf)
			t.CheckError(tc.wantPreHooksErr, err)

			err = pb.PostBuild(ctx, buf)
			t.CheckError(tc.wantPostHooksErr, err)

			if !tc.wantPreHooksErr && !tc.wantPostHooksErr {
				t.CheckDeepEqual(tc.expected, buf.Bytes())
			}
		})
	}
}

type mockPipelineBuilder struct{}

func (m *mockPipelineBuilder) PreBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact) build.ArtifactBuilder {
	return nil
}

func (m *mockPipelineBuilder) PostBuild(ctx context.Context, out io.Writer) error { return nil }

func (m *mockPipelineBuilder) Concurrency() *int { return nil }

func (m *mockPipelineBuilder) Prune(context.Context, io.Writer) error { return nil }

func (m *mockPipelineBuilder) PushImages() bool { return false }

func (m *mockPipelineBuilder) SupportedPlatforms() platform.Matcher { return platform.All }
