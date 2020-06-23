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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInSequence(t *testing.T) {
	tests := []struct {
		description       string
		buildArtifact     artifactBuilder
		tags              tag.ImageTags
		expectedArtifacts []Artifact
		expectedOut       string
		shouldErr         bool
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
			expectedArtifacts: []Artifact{
				{ImageName: "skaffold/image1", Tag: "skaffold/image1:v0.0.1@sha256:abac"},
				{ImageName: "skaffold/image2", Tag: "skaffold/image2:v0.0.2@sha256:abac"},
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
			artifacts := []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			}

			got, err := InSequence(context.Background(), out, test.tags, artifacts, test.buildArtifact)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedArtifacts, got)
			t.CheckDeepEqual(test.expectedOut, out.String())
		})
	}
}

func TestInSequenceResultsOrder(t *testing.T) {
	tests := []struct {
		description string
		images      []string
		expected    []Artifact
		shouldErr   bool
	}{
		{
			description: "shd concatenate the tag",
			images:      []string{"a", "b", "c", "d"},
			expected: []Artifact{
				{ImageName: "a", Tag: "a:a"},
				{ImageName: "b", Tag: "b:ab"},
				{ImageName: "c", Tag: "c:abc"},
				{ImageName: "d", Tag: "d:abcd"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := ioutil.Discard
			initializeEvents()
			artifacts := make([]*latest.Artifact, len(test.images))
			tags := tag.ImageTags{}
			for i, image := range test.images {
				artifacts[i] = &latest.Artifact{
					ImageName: image,
				}
				tags[image] = image
			}
			builder := concatTagger{}

			got, err := InSequence(context.Background(), out, tags, artifacts, builder.doBuild)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

// concatTagger builder sums all the numbers
type concatTagger struct {
	tag string
}

// doBuild calculate the tag based by concatinating the tag values for artifact
// builds seen so far. It mimics artifact dependency where the next build result
// depends on the previous build result.
func (t *concatTagger) doBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	t.tag += tag
	return fmt.Sprintf("%s:%s", artifact.ImageName, t.tag), nil
}

func initializeEvents() {
	event.InitializeState(latest.Pipeline{
		Deploy: latest.DeployConfig{},
		Build: latest.BuildConfig{
			BuildType: latest.BuildType{
				LocalBuild: &latest.LocalBuild{},
			},
		},
	},
		"temp",
		true,
		true,
		true)
}
