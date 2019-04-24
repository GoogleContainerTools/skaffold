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
	"context"
	"errors"

	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPreBuiltImagesBuilder(t *testing.T) {
	type testResult struct {
		buildResult Result
		shouldErr   bool
	}

	var tests = []struct {
		description string
		images      []string
		artifacts   []*latest.Artifact
		expected    []testResult
	}{
		{
			description: "images in same order",
			images: []string{
				"skaffold/image1:tag1",
				"skaffold/image2:tag2",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image1"},
						Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
					},
				},
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image2"},
						Result: Artifact{ImageName: "skaffold/image2", Tag: "skaffold/image2:tag2"},
					},
				},
			},
		},
		{
			description: "images in reverse order",
			images: []string{
				"skaffold/image2:tag2",
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image1"},
						Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
					},
				},
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image2"},
						Result: Artifact{ImageName: "skaffold/image2", Tag: "skaffold/image2:tag2"},
					},
				},
			},
		},
		{
			description: "missing image",
			images: []string{
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			},
			expected: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image1"},
						Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
					},
				},
				{
					shouldErr: true,
					buildResult: Result{
						Error: errors.New("unable to find image tag for skaffold/image2"),
					},
				},
			},
		},
		{
			// Should we support that? It is used in kustomize example.
			description: "additional image",
			images: []string{
				"busybox:1",
				"skaffold/image1:tag1",
			},
			artifacts: []*latest.Artifact{
				{ImageName: "skaffold/image1"},
			},
			expected: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image1"},
						Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
					},
				},
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "busybox"},
						Result: Artifact{ImageName: "busybox", Tag: "busybox:1"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			builder := NewPreBuiltImagesBuilder(&runcontext.RunContext{
				Opts: &config.SkaffoldOptions{
					PreBuiltImages: test.images,
				},
			})

			bRes, _ := builder.Build(context.Background(), ioutil.Discard, nil, test.artifacts)

			for i, r := range test.expected {
				if r.shouldErr {
					testutil.CheckError(t, true, r.buildResult.Error)
				} else {
					testutil.CheckDeepEqual(t, r.buildResult, bRes[i])
				}
			}
		})
	}
}
