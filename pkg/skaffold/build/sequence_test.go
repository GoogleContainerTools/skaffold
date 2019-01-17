/*
Copyright 2018 The Skaffold Authors

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

type MockTagger struct {
	Out string
	Err error
}

func (f *MockTagger) GenerateFullyQualifiedImageName(workingDir string, imageName string) (string, error) {
	return f.Out, f.Err
}

func (f *MockTagger) Labels() map[string]string {
	return map[string]string{}
}

func TestInSequence(t *testing.T) {
	var tests = []struct {
		description       string
		buildArtifact     artifactBuilder
		expectedArtifacts []Artifact
		expectedOut       string
		shouldErr         bool
	}{
		{
			description: "build fails",
			buildArtifact: func(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
				return "", fmt.Errorf("build fails")
			},
			expectedArtifacts: nil,
			expectedOut:       "Building [skaffold/image1]...\n",
			shouldErr:         true,
		},
		{
			description: "build succeeds",
			buildArtifact: func(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
				return "v0.0.1", nil
			},
			expectedArtifacts: []Artifact{
				{ImageName: "skaffold/image1", Tag: "v0.0.1"},
				{ImageName: "skaffold/image2", Tag: "v0.0.1"},
			},
			expectedOut: "Building [skaffold/image1]...\nBuilding [skaffold/image2]...\n",
			shouldErr:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			out := new(bytes.Buffer)
			artifacts := []*latest.Artifact{
				{ImageName: "skaffold/image1"},
				{ImageName: "skaffold/image2"},
			}
			tagger := &MockTagger{Out: ""}

			got, err := InSequence(context.Background(), out, tagger, artifacts, test.buildArtifact)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedArtifacts, got)
			testutil.CheckDeepEqual(t, test.expectedOut, out.String())
		})
	}
}
