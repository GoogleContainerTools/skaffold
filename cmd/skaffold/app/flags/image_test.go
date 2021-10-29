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

package flags

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewEmptyImage(t *testing.T) {
	flag := NewEmptyImages("empty image")
	expectedFlag := &Images{images: []image{}, usage: "empty image"}
	if flag.String() != expectedFlag.String() {
		t.Errorf("expected %s, actual %s", expectedFlag, flag)
	}
}

func TestImagesFlagSet(t *testing.T) {
	// These tests only check Set() with a single value.
	tests := []struct {
		description      string
		setValue         string
		shouldErr        bool
		expectedArtifact graph.Artifact
	}{
		{
			description: "set on correct value return right artifact with tag",
			setValue:    "gcr.io/test/test-image:test",
			expectedArtifact: graph.Artifact{
				ImageName: "gcr.io/test/test-image",
				Tag:       "gcr.io/test/test-image:test",
			},
		},
		{
			description: "set on correct value return right artifact without tag",
			setValue:    "gcr.io/test/test-image",
			expectedArtifact: graph.Artifact{
				ImageName: "gcr.io/test/test-image",
				Tag:       "gcr.io/test/test-image",
			},
		},
		{
			description: "set on correct value return right artifact without digest",
			setValue:    "gcr.io/test/test-image@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedArtifact: graph.Artifact{
				ImageName: "gcr.io/test/test-image",
				Tag:       "gcr.io/test/test-image@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			},
		},
		{
			description: "set with docker name",
			setValue:    "docker-image-value",
			expectedArtifact: graph.Artifact{
				ImageName: "docker-image-value",
				Tag:       "docker-image-value",
			},
		},
		{
			description: "set with name=tag",
			setValue:    "name=tag",
			expectedArtifact: graph.Artifact{
				ImageName: "name",
				Tag:       "tag",
			},
		},
		{
			description: "set errors with invalid docker name",
			setValue:    "docker_:!",
			shouldErr:   true,
		},
		{
			description: "set errors with empty image name",
			setValue:    "",
			shouldErr:   true,
		},
		{
			description: "set errors with invalid docker name-tag pair",
			setValue:    "name=docker_:!",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			flag := NewEmptyImages("input image name")

			err := flag.Set(test.setValue)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				image := flag.images[0]
				// Test flag value is set to new value and expected Artifact
				t.CheckDeepEqual(test.expectedArtifact, *image.artifact)
				t.CheckDeepEqual(test.setValue, image.name)
			}
		})
	}
}

func TestImagesSetCSV(t *testing.T) {
	flag := NewEmptyImages("input image flag")
	flag.Set("test-image1:test,test-image2:latest")
	testutil.CheckDeepEqual(t, 2, len(flag.GetSlice()))
	testutil.CheckDeepEqual(t, "test-image1:test,test-image2:latest", flag.String())

	// check repeated calls to Set accumulate values.  This is backwards compatibility
	// with the old behaviour that required `-i` for each image.
	flag.Set("test-image3:test")
	testutil.CheckDeepEqual(t, 3, len(flag.GetSlice()))
	testutil.CheckDeepEqual(t, "test-image1:test,test-image2:latest,test-image3:test", flag.String())
}

func TestImagesString(t *testing.T) {
	flag := NewEmptyImages("input image flag")
	flag.Append("gcr.io/test/test-image:test")
	flag.Append("gcr.io/test/test-image-1:test")
	testutil.CheckDeepEqual(t, "gcr.io/test/test-image:test,gcr.io/test/test-image-1:test", flag.String())

	flag.SetNil()
	testutil.CheckDeepEqual(t, "", flag.String())

	flag.Set("gcr.io/test/test-image:test,gcr.io/test/test-image-1:test,name=tag")
	testutil.CheckDeepEqual(t, "gcr.io/test/test-image:test,gcr.io/test/test-image-1:test,name=tag", flag.String())
}

func TestImagesType(t *testing.T) {
	flag := NewEmptyImages("input docker image name")
	expectedFlagType := "*flags.Images"
	if flag.Type() != expectedFlagType {
		t.Errorf("Flag returned wrong type. Expected %s, Actual %s", expectedFlagType, flag.Type())
	}
}

func TestConvertToArtifact(t *testing.T) {
	tests := []struct {
		description string
		image       string
		expected    *graph.Artifact
		shouldErr   bool
	}{
		{
			description: "valid image",
			image:       "skaffold/image1:tag1",
			expected:    &graph.Artifact{ImageName: "skaffold/image1", Tag: "skaffold/image1:tag1"},
		},
		{
			description: "test invalid artifact",
			image:       "busybox:1$",
			shouldErr:   true,
		},
		{
			description: "empty artifact",
			image:       "",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			bRes, err := convertImageToArtifact(test.image)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, bRes)
		})
	}
}
