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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInSequence(t *testing.T) {
	var tests = []struct {
		description     string
		buildArtifact   artifactBuilder
		tags            tag.ImageTags
		expectedResults []Result
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
			expectedResults: []Result{
				{
					Target: latest.Artifact{ImageName: "skaffold/image1"},
					Result: Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:v0.0.1@sha256:abac"},
				},
				{
					Target: latest.Artifact{ImageName: "skaffold/image2"},
					Result: Artifact{ImageName: "skaffold/image2", Tag: "skaffold/image2:v0.0.2@sha256:abac"},
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
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
			expectedResults: []Result{
				{
					Target: latest.Artifact{
						ImageName: "skaffold/image1",
					},
					Error: errors.New("building [skaffold/image1]: build fails"),
				},
				{
					Target: latest.Artifact{
						ImageName: "skaffold/image2",
					},
					Error: errors.New("unable to find tag for image skaffold/image2"),
				},
			},
		},
		{
			description: "tag not found",
			tags:        tag.ImageTags{},
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
			expectedResults: []Result{
				{
					Target: latest.Artifact{
						ImageName: "skaffold/image1",
					},
					Error: errors.New("unable to find tag for image skaffold/image1"),
				},
				{
					Target: latest.Artifact{
						ImageName: "skaffold/image2",
					},
					Error: errors.New("unable to find tag for image skaffold/image2"),
				},
			},
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

			ch := make(chan Result)
			results, err := InSequence(context.Background(), out, test.tags, artifacts, test.buildArtifact, ch)
			// Wait for all results
			for i := 0; i < len(artifacts); i++ {
				<-ch
			}
			testutil.CheckError(t, test.shouldErr, err)

			fmt.Fprintf(os.Stdout, "final build results: %+v\n", results)

			CheckResultsPerArtifact(t, test.expectedResults, results)

			testutil.CheckDeepEqual(t, test.expectedOut, out.String())
		})
	}
}

func TestInSequenceResultsSeen(t *testing.T) {
	var tests = []struct {
		description   string
		images        []string
		expectedOrder []Result
	}{
		{
			description: "shd see results sequentially in order of input",
			images:      []string{"four", "one", "eight", "two"},
			expectedOrder: []Result{
				{
					Target: latest.Artifact{ImageName: "four"},
					Result: Artifact{ImageName: "four", Tag: "four:tag@sha256:abac"},
				},
				{
					Target: latest.Artifact{ImageName: "one"},
					Result: Artifact{ImageName: "one", Tag: "one:tag@sha256:abac"},
				},
				{
					Target: latest.Artifact{ImageName: "eight"},
					Result: Artifact{ImageName: "eight", Tag: "eight:tag@sha256:abac"},
				},
				{
					Target: latest.Artifact{ImageName: "two"},
					Result: Artifact{ImageName: "two", Tag: "two:tag@sha256:abac"},
				},
			},
		},
		// Add test when artifact has an error
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			out := ioutil.Discard
			artifacts := make([]*latest.Artifact, len(test.images))
			tags := tag.ImageTags{}
			for i, image := range test.images {
				artifacts[i] = &latest.Artifact{
					ImageName: image,
				}
				tags[image] = fmt.Sprintf("%s:tag", image)
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

			ch := make(chan Result, len(test.images))
			InSequence(context.Background(), out, tags, artifacts, StaggerBuilder, ch)
			actualOrder := make([]Result, len(test.images))
			// Wait for all results
			for i := 0; i < len(artifacts); i++ {
				actualOrder[i] = <-ch
			}
			CheckResultsPerArtifact(t, test.expectedOrder, actualOrder)
		})
	}
}
