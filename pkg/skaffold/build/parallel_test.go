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
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInParallel(t *testing.T) {
	var tests = []struct {
		description     string
		buildArtifact   artifactBuilder
		tags            tag.ImageTags
		expectedResults []testResult
		expectedOut     string
		shouldErr       bool
	}{
		{
			description: "build succeeds",
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				return fmt.Sprintf("%s@sha256:abac", tag), nil
			},
			tags: tag.ImageTags{
				"skaffold/image1": "skaffold/image1:v0.0.1",
				"skaffold/image2": "skaffold/image2:v0.0.2",
			},
			expectedResults: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image1"},
						Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:v0.0.1@sha256:abac"},
					},
				},
				{
					buildResult: Result{
						Target: latest.Artifact{ImageName: "skaffold/image2"},
						Result: Artifact{ImageName: "skaffold/image2", Tag: "skaffold/image2:v0.0.2@sha256:abac"},
					},
				},
			},
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
		},
		{
			description: "build fails",
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
				return "", fmt.Errorf("build fails")
			},
			tags: tag.ImageTags{
				"skaffold/image1": "",
			},
			expectedResults: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{
							ImageName: "skaffold/image1",
						},
						Error: errors.New("build fails"),
					},
					shouldErr: true,
				},
				{
					buildResult: Result{
						Target: latest.Artifact{
							ImageName: "skaffold/image2",
						},
						Error: errors.New("building [skaffold/image2]: unable to find tag for image"),
					},
					shouldErr: true,
				},
			},
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
		},
		{
			description: "tag not found",
			tags:        tag.ImageTags{},
			expectedResults: []testResult{
				{
					buildResult: Result{
						Target: latest.Artifact{
							ImageName: "skaffold/image1",
						},
						Error: errors.New("building [skaffold/image1]: unable to find tag for image"),
					},
					shouldErr: true,
				},
				{
					buildResult: Result{
						Target: latest.Artifact{
							ImageName: "skaffold/image2",
						},
						Error: errors.New("building [skaffold/image2]: unable to find tag for image"),
					},
					shouldErr: true,
				},
			},
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			out := new(bytes.Buffer)
			artifacts := []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			}
			cfg := latest.BuildConfig{
				BuildType: latest.BuildType{
					LocalBuild: &latest.LocalBuild{},
				},
			}
			event.InitializeState(&runcontext.RunContext{
				Cfg: &latest.Pipeline{
					Build: cfg,
				},
				Opts: &config.SkaffoldOptions{},
			})

			buildResultChannels, err := InParallel(context.Background(), out, test.tags, artifacts, test.buildArtifact)
			testutil.CheckError(t, test.shouldErr, err)

			res := CollectResultsFromChannels(buildResultChannels)

			// build results are returned in a list, of which we can't guarantee order.
			// loop through the expected results, and find the matching build result by target artifact.
			found := false
			for _, testRes := range test.expectedResults {
				for _, buildRes := range res {
					if buildRes.Target.ImageName == testRes.buildResult.Target.ImageName {
						found = true
						// the embedded error in the build result contains a stack trace which we can't reproduce.
						// directly compare the fields of the build result and optional error.
						testutil.CheckError(t, testRes.shouldErr, buildRes.Error)
						if testRes.shouldErr {
							testutil.CheckDeepEqual(t, testRes.buildResult.Error.Error(), buildRes.Error.Error())
						}
						testutil.CheckDeepEqual(t, testRes.buildResult.Target, buildRes.Target)
						testutil.CheckDeepEqual(t, testRes.buildResult.Result, buildRes.Result)
					}
				}
				if !found {
					t.Errorf("expected result %+v not found in build results", testRes)
				}
				found = false
			}

			testutil.CheckDeepEqual(t, test.expectedOut, out.String())
		})
	}
}
