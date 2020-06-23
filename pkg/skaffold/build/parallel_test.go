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

package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetBuild(t *testing.T) {
	tests := []struct {
		description   string
		buildArtifact artifactBuilder
		tags          tag.ImageTags
		expectedTag   string
		expectedOut   string
		shouldErr     bool
	}{
		{
			description: "build succeeds",
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				out.Write([]byte("build succeeds"))
				return fmt.Sprintf("%s@sha256:abac", tag), nil
			},
			tags: tag.ImageTags{
				"skaffold/image1": "skaffold/image1:v0.0.1",
				"skaffold/image2": "skaffold/image2:v0.0.2",
			},
			expectedTag: "skaffold/image1:v0.0.1@sha256:abac",
			expectedOut: "Building [skaffold/image1]...\nbuild succeeds",
		},
		{
			description: "build fails",
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				return "", fmt.Errorf("build fails")
			},
			tags: tag.ImageTags{
				"skaffold/image1": "",
			},
			expectedOut: "Building [skaffold/image1]...\n",
			shouldErr:   true,
		},
		{
			description: "tag not found",
			tags:        tag.ImageTags{},
			expectedOut: "Building [skaffold/image1]...\n",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := new(bytes.Buffer)

			artifact := &latest.Artifact{ImageName: "skaffold/image1"}
			got, err := getBuildResult(context.Background(), out, test.tags, artifact, test.buildArtifact)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedTag, got)
			t.CheckDeepEqual(test.expectedOut, out.String())
		})
	}
}

func TestCollectResults(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		expected    []Artifact
		results     map[string]interface{}
		shouldErr   bool
	}{
		{
			description: "all builds completely successfully",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:v0.0.1@sha256:abac"},
				{ImageName: "skaffold/image2", Tag: "skaffold/image2:v0.0.2@sha256:abac"},
			},
			results: map[string]interface{}{
				"skaffold/image1": Artifact{
					ImageName: "skaffold/image1",
					Tag:       "skaffold/image1:v0.0.1@sha256:abac",
				},
				"skaffold/image2": Artifact{
					ImageName: "skaffold/image2",
					Tag:       "skaffold/image2:v0.0.2@sha256:abac",
				},
			},
		},
		{
			description: "first build errors",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: nil,
			results: map[string]interface{}{
				"skaffold/image1": fmt.Errorf("Could not build image skaffold/image1"),
				"skaffold/image2": Artifact{
					ImageName: "skaffold/image2",
					Tag:       "skaffold/image2:v0.0.2@sha256:abac",
				},
			},
			shouldErr: true,
		},
		{
			description: "arbitrary image build failure",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
				{ImageName: "skaffold/image3"},
			},
			expected: nil,
			results: map[string]interface{}{
				"skaffold/image1": Artifact{
					ImageName: "skaffold/image1",
					Tag:       "skaffold/image1:v0.0.1@sha256:abac",
				},
				"skaffold/image2": fmt.Errorf("Could not build image skaffold/image1"),
				"skaffold/image3": Artifact{
					ImageName: "skaffold/image3",
					Tag:       "skaffold/image3:v0.0.1@sha256:abac",
				},
			},
			shouldErr: true,
		},
		{
			description: "no build result produced for a build",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: nil,
			results: map[string]interface{}{
				"skaffold/image1": Artifact{
					ImageName: "skaffold/image1:v0.0.1@sha256:abac",
					Tag:       "skaffold/image1:v0.0.1@sha256:abac",
				},
			},
			shouldErr: true,
		},
		{
			description: "build produced an incorrect value type",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: nil,
			results: map[string]interface{}{
				"skaffold/image1": 1,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			outputs := setUpChannels(len(test.artifacts))
			resultMap := new(sync.Map)
			for k, v := range test.results {
				resultMap.Store(k, v)
			}

			got, err := collectResults(ioutil.Discard, test.artifacts, resultMap, outputs)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

func TestInParallel(t *testing.T) {
	tests := []struct {
		description string
		buildFunc   artifactBuilder
		expected    string
	}{
		{
			description: "short and nice build log",
			expected:    "Building [skaffold/image1]...\nshort\nBuilding [skaffold/image2]...\nshort\n",
			buildFunc: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				out.Write([]byte("short"))
				return fmt.Sprintf("%s:tag", artifact.ImageName), nil
			},
		},
		{
			description: "long build log gets printed correctly",
			expected: `Building [skaffold/image1]...
This is a long string more than 10 bytes.
And new lines
Building [skaffold/image2]...
This is a long string more than 10 bytes.
And new lines
`,
			buildFunc: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				out.Write([]byte("This is a long string more than 10 bytes.\nAnd new lines"))
				return fmt.Sprintf("%s:tag", artifact.ImageName), nil
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := new(bytes.Buffer)
			artifacts := []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			}
			tags := tag.ImageTags{
				"skaffold/image1": "skaffold/image1:v0.0.1",
				"skaffold/image2": "skaffold/image2:v0.0.2",
			}
			initializeEvents()

			InParallel(context.Background(), out, tags, artifacts, test.buildFunc, 0)

			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}

func TestInParallelConcurrency(t *testing.T) {
	tests := []struct {
		artifacts      int
		limit          int
		maxConcurrency int
	}{
		{
			artifacts:      10,
			limit:          0, // default - no limit
			maxConcurrency: 10,
		},
		{
			artifacts:      50,
			limit:          1,
			maxConcurrency: 1,
		},
		{
			artifacts:      50,
			limit:          10,
			maxConcurrency: 10,
		},
	}
	for _, test := range tests {
		testutil.Run(t, fmt.Sprintf("%d artifacts, max concurrency=%d", test.artifacts, test.limit), func(t *testutil.T) {
			var artifacts []*latest.Artifact
			tags := tag.ImageTags{}

			for i := 0; i < test.artifacts; i++ {
				imageName := fmt.Sprintf("skaffold/image%d", i)
				tag := fmt.Sprintf("skaffold/image%d:tag", i)

				artifacts = append(artifacts, &latest.Artifact{ImageName: imageName})
				tags[imageName] = tag
			}

			var actualConcurrency int32

			builder := func(_ context.Context, _ io.Writer, _ *latest.Artifact, tag string) (string, error) {
				if atomic.AddInt32(&actualConcurrency, 1) > int32(test.maxConcurrency) {
					return "", fmt.Errorf("only %d build can run at a time", test.maxConcurrency)
				}
				time.Sleep(5 * time.Millisecond)
				atomic.AddInt32(&actualConcurrency, -1)

				return tag, nil
			}

			initializeEvents()
			results, err := InParallel(context.Background(), ioutil.Discard, tags, artifacts, builder, test.limit)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.artifacts, len(results))
		})
	}
}

func TestInParallelForArgs(t *testing.T) {
	tests := []struct {
		description   string
		inSeqFunc     func(context.Context, io.Writer, tag.ImageTags, []*latest.Artifact, artifactBuilder) ([]Artifact, error)
		buildArtifact artifactBuilder
		artifactLen   int
		expected      []Artifact
	}{
		{
			description: "runs in sequence for 1 artifact",
			inSeqFunc: func(context.Context, io.Writer, tag.ImageTags, []*latest.Artifact, artifactBuilder) ([]Artifact, error) {
				return []Artifact{{ImageName: "singleArtifact", Tag: "one"}}, nil
			},
			artifactLen: 1,
			expected:    []Artifact{{ImageName: "singleArtifact", Tag: "one"}},
		},
		{
			description: "runs in parallel for 2 artifacts",
			buildArtifact: func(_ context.Context, _ io.Writer, _ *latest.Artifact, tag string) (string, error) {
				return tag, nil
			},
			artifactLen: 2,
			expected: []Artifact{
				{ImageName: "artifact1", Tag: "artifact1@tag1"},
				{ImageName: "artifact2", Tag: "artifact2@tag2"},
			},
		},
		{
			description: "runs in parallel should return for 0 artifacts",
			artifactLen: 0,
			expected:    nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifacts := make([]*latest.Artifact, test.artifactLen)
			tags := tag.ImageTags{}
			for i := 0; i < test.artifactLen; i++ {
				a := fmt.Sprintf("artifact%d", i+1)
				artifacts[i] = &latest.Artifact{ImageName: a}
				tags[a] = fmt.Sprintf("%s@tag%d", a, i+1)
			}
			if test.inSeqFunc != nil {
				t.Override(&runInSequence, test.inSeqFunc)
			}
			initializeEvents()
			actual, _ := InParallel(context.Background(), ioutil.Discard, tags, artifacts, test.buildArtifact, 0)

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func setUpChannels(n int) []chan string {
	outputs := make([]chan string, n)
	for i := 0; i < n; i++ {
		outputs[i] = make(chan string, 10)
		close(outputs[i])
	}
	return outputs
}
