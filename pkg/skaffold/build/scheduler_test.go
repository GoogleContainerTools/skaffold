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

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetBuild(t *testing.T) {
	tests := []struct {
		description   string
		buildArtifact ArtifactBuilder
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
			got, err := performBuild(context.Background(), out, test.tags, artifact, test.buildArtifact)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedTag, got)
			t.CheckDeepEqual(test.expectedOut, out.String())
		})
	}
}

func TestFormatResults(t *testing.T) {
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
				"skaffold/image1": "skaffold/image1:v0.0.1@sha256:abac",
				"skaffold/image2": "skaffold/image2:v0.0.2@sha256:abac",
			},
		},
		{
			description: "no build result produced for a build",
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: nil,
			results: map[string]interface{}{
				"skaffold/image1": "skaffold/image1:v0.0.1@sha256:abac",
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			m := new(sync.Map)
			for k, v := range test.results {
				m.Store(k, v)
			}
			results := &artifactStoreImpl{m: m}
			got, err := results.GetArtifacts(test.artifacts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

func TestInOrder(t *testing.T) {
	tests := []struct {
		description string
		buildFunc   ArtifactBuilder
		expected    string
	}{
		{
			description: "short and nice build log",
			expected:    "Building 2 artifacts in parallel\nBuilding [skaffold/image1]...\nshort\nBuilding [skaffold/image2]...\nshort\n",
			buildFunc: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				out.Write([]byte("short"))
				return fmt.Sprintf("%s:tag", artifact.ImageName), nil
			},
		},
		{
			description: "long build log gets printed correctly",
			expected: `Building 2 artifacts in parallel
Building [skaffold/image1]...
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
				{ImageName: "skaffold/image2", Dependencies: []*latest.ArtifactDependency{{ImageName: "skaffold/image1"}}},
			}
			tags := tag.ImageTags{
				"skaffold/image1": "skaffold/image1:v0.0.1",
				"skaffold/image2": "skaffold/image2:v0.0.2",
			}
			initializeEvents()

			InOrder(context.Background(), out, tags, artifacts, test.buildFunc, 0, NewArtifactStore())

			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}

func TestInOrderConcurrency(t *testing.T) {
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
			results, err := InOrder(context.Background(), ioutil.Discard, tags, artifacts, builder, test.limit, NewArtifactStore())

			t.CheckNoError(err)
			t.CheckDeepEqual(test.artifacts, len(results))
		})
	}
}

func TestInOrderForArgs(t *testing.T) {
	tests := []struct {
		description   string
		buildArtifact ArtifactBuilder
		artifactLen   int
		concurrency   int
		dependency    map[int][]int
		expected      []Artifact
		err           error
	}{
		{
			description: "runs in parallel for 2 artifacts with no dependency",
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
			description: "runs in parallel for 5 artifacts with dependencies",
			buildArtifact: func(_ context.Context, _ io.Writer, _ *latest.Artifact, tag string) (string, error) {
				return tag, nil
			},
			dependency: map[int][]int{
				0: {2, 3},
				1: {3},
				2: {1},
				3: {4},
			},
			artifactLen: 5,
			expected: []Artifact{
				{ImageName: "artifact1", Tag: "artifact1@tag1"},
				{ImageName: "artifact2", Tag: "artifact2@tag2"},
				{ImageName: "artifact3", Tag: "artifact3@tag3"},
				{ImageName: "artifact4", Tag: "artifact4@tag4"},
				{ImageName: "artifact5", Tag: "artifact5@tag5"},
			},
		},
		{
			description: "runs with max concurrency of 2 for 5 artifacts with dependencies",
			buildArtifact: func(_ context.Context, _ io.Writer, _ *latest.Artifact, tag string) (string, error) {
				return tag, nil
			},
			dependency: map[int][]int{
				0: {2, 3},
				1: {3},
				2: {1},
				3: {4},
			},
			artifactLen: 5,
			concurrency: 2,
			expected: []Artifact{
				{ImageName: "artifact1", Tag: "artifact1@tag1"},
				{ImageName: "artifact2", Tag: "artifact2@tag2"},
				{ImageName: "artifact3", Tag: "artifact3@tag3"},
				{ImageName: "artifact4", Tag: "artifact4@tag4"},
				{ImageName: "artifact5", Tag: "artifact5@tag5"},
			},
		},
		{
			description: "runs in parallel should return for 0 artifacts",
			artifactLen: 0,
			expected:    nil,
		},
		{
			description: "build fails for artifacts without dependencies",
			buildArtifact: func(c context.Context, _ io.Writer, a *latest.Artifact, tag string) (string, error) {
				if a.ImageName == "artifact2" {
					return "", fmt.Errorf(`some error occurred while building "artifact2"`)
				}
				select {
				case <-c.Done():
					return "", c.Err()
				case <-time.After(5 * time.Second):
					return tag, nil
				}
			},
			artifactLen: 5,
			expected:    nil,
			err:         fmt.Errorf(`some error occurred while building "artifact2"`),
		},
		{
			description: "build fails for artifacts with dependencies",
			buildArtifact: func(_ context.Context, _ io.Writer, a *latest.Artifact, tag string) (string, error) {
				if a.ImageName == "artifact2" {
					return "", fmt.Errorf(`some error occurred while building "artifact2"`)
				}
				return tag, nil
			},
			dependency: map[int][]int{
				0: {1},
				1: {2},
				2: {3},
				3: {4},
			},
			artifactLen: 5,
			expected:    nil,
			err:         fmt.Errorf(`some error occurred while building "artifact2"`),
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

			setDependencies(artifacts, test.dependency)
			initializeEvents()
			actual, err := InOrder(context.Background(), ioutil.Discard, tags, artifacts, test.buildArtifact, test.concurrency, NewArtifactStore())

			t.CheckDeepEqual(test.expected, actual)
			t.CheckDeepEqual(test.err, err, cmp.Comparer(errorsComparer))
		})
	}
}

// setDependencies constructs a graph of artifact dependencies using the map as an adjacency list representation of indices in the artifacts array.
// For example:
// m = {
//    0 : {1, 2},
//    2 : {3},
//}
// implies that a[0] artifact depends on a[1] and a[2]; and a[2] depends on a[3].
func setDependencies(a []*latest.Artifact, d map[int][]int) {
	for k, dep := range d {
		for i := range dep {
			a[k].Dependencies = append(a[k].Dependencies, &latest.ArtifactDependency{
				ImageName: a[dep[i]].ImageName,
			})
		}
	}
}

func initializeEvents() {
	pipes := []latest.Pipeline{{
		Deploy: latest.DeployConfig{},
		Build: latest.BuildConfig{
			BuildType: latest.BuildType{
				LocalBuild: &latest.LocalBuild{},
			},
		},
	}}
	event.InitializeState(pipes, "temp", true, true, true)
}

func errorsComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Error() == b.Error()
}
