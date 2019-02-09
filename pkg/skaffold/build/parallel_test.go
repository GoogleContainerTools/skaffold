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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInParallel(t *testing.T) {
	var tests = []struct {
		description       string
		buildArtifact     artifactBuilder
		tags              tag.ImageTags
		expectedArtifacts []Artifact
		expectedOut       string
		shouldErr         bool
	}{
		{
			description: "build succeeds",
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, ConfigurationRetriever, error) {
				return fmt.Sprintf("%s@sha256:abac", tag), nil, nil
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
			buildArtifact: func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, ConfigurationRetriever, error) {
				return "", nil, fmt.Errorf("build fails")
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
		t.Run(test.description, func(t *testing.T) {
			out := new(bytes.Buffer)
			artifacts := []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			}

			got, err := InParallel(context.Background(), out, test.tags, artifacts, test.buildArtifact)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedArtifacts, got)
			testutil.CheckDeepEqual(t, test.expectedOut, out.String())
		})
	}
}
