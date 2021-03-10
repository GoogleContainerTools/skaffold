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

package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetArtifacts(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		fromFile    []build.Artifact
		fromCLI     []build.Artifact
		expected    []build.Artifact
		customTag   string
		shouldErr   bool
	}{
		{
			description: "no artifact",
			artifacts:   nil,
			fromFile:    nil,
			fromCLI:     nil,
			expected:    []build.Artifact(nil),
		},
		{
			description: "from file",
			artifacts:   []*latest.Artifact{{ImageName: "image"}},
			fromFile:    []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
			fromCLI:     nil,
			expected:    []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
		},
		{
			description: "from CLI",
			artifacts:   []*latest.Artifact{{ImageName: "image"}},
			fromFile:    nil,
			fromCLI:     []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
			expected:    []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
		},
		{
			description: "one from file, one from CLI",
			artifacts:   []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			fromFile:    []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}},
			fromCLI:     []build.Artifact{{ImageName: "image2", Tag: "image2:tag"}},
			expected:    []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}, {ImageName: "image2", Tag: "image2:tag"}},
		},
		{
			description: "file takes precedence on CLI",
			artifacts:   []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			fromFile:    []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}, {ImageName: "image2", Tag: "image2:tag"}},
			fromCLI:     []build.Artifact{{ImageName: "image1", Tag: "image1:ignored"}},
			expected:    []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}, {ImageName: "image2", Tag: "image2:tag"}},
		},
		{
			description: "provide tag for non-artifact",
			artifacts:   []*latest.Artifact{},
			fromCLI:     []build.Artifact{{ImageName: "busybox", Tag: "busybox:v1"}},
			expected:    []build.Artifact{{ImageName: "busybox", Tag: "busybox:v1"}},
		},
		{
			description: "missing tag",
			artifacts:   []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			fromFile:    nil,
			fromCLI:     nil,
			shouldErr:   true,
		},
		{
			description: "override tag",
			artifacts:   []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			fromFile:    []build.Artifact{{ImageName: "image1", Tag: "image1:tag"}},
			fromCLI:     []build.Artifact{{ImageName: "image2", Tag: "image2:tag"}},
			expected:    []build.Artifact{{ImageName: "image1", Tag: "image1:test"}, {ImageName: "image2", Tag: "image2:test"}},
			customTag:   "test",
		},
		{
			description: "override missing tag",
			artifacts:   []*latest.Artifact{{ImageName: "image1"}, {ImageName: "image2"}},
			fromFile:    nil,
			fromCLI:     nil,
			expected:    []build.Artifact{{ImageName: "image1", Tag: "image1:test"}, {ImageName: "image2", Tag: "image2:test"}},
			customTag:   "test",
		},
		{
			description: "apply tags to no artifacts",
			artifacts:   []*latest.Artifact{},
			fromFile:    nil,
			fromCLI:     nil,
			expected:    []build.Artifact(nil),
			customTag:   "test",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.customTag != "" {
				t.Override(&opts.CustomTag, test.customTag)
			}

			artifacts, err := mergeBuildArtifacts(test.fromFile, test.fromCLI, test.artifacts)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, artifacts)
		})
	}
}

func Test_getBuildArtifactsAndSetTags(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []build.Artifact
		expected    []build.Artifact
		defaultRepo string
		shouldErr   bool
	}{
		{
			description: "no artifact without default-repo",
			artifacts:   nil,
			expected:    []build.Artifact(nil),
		},
		{
			description: "single artifact without default-repo",
			artifacts:   []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
			expected:    []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
		},
		{
			description: "multiple artifacts without default-repo",
			artifacts: []build.Artifact{
				{ImageName: "image1", Tag: "image1:tag"},
				{ImageName: "image1", Tag: "image1:tag"},
			},
			expected: []build.Artifact{
				{ImageName: "image1", Tag: "image1:tag"},
				{ImageName: "image1", Tag: "image1:tag"},
			},
		},
		{
			description: "single artifact with default-repo",
			artifacts:   []build.Artifact{{ImageName: "image", Tag: "image:tag"}},
			expected:    []build.Artifact{{ImageName: "image", Tag: "example.com/test-repo/image:tag"}},
			defaultRepo: "example.com/test-repo",
		},
		{
			description: "multiple artifacts with default-repo",
			artifacts: []build.Artifact{
				{ImageName: "image1", Tag: "image1:tag"},
				{ImageName: "image1", Tag: "image1:tag"},
			},
			expected: []build.Artifact{
				{ImageName: "image1", Tag: "example.com/test-repo/image1:tag"},
				{ImageName: "image1", Tag: "example.com/test-repo/image1:tag"},
			},
			defaultRepo: "example.com/test-repo",
		},
		{
			description: "multiple artifacts with erring default-repo",
			artifacts: []build.Artifact{
				{ImageName: "image1", Tag: "image1:tag"},
				{ImageName: "image1", Tag: "image1:tag"},
			},
			expected:    []build.Artifact(nil),
			defaultRepo: "example.com/test-repo",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifacts, err := applyDefaultRepoToArtifacts(test.artifacts, func(s string) (string, error) {
				if test.shouldErr {
					// this seems counter-intuitive that we explicitly return an error when shouldErr is true,
					// however this function is a callback, the test is ensuring the error from the callback is handled
					// correctly
					return "", errors.New("error")
				}

				if test.defaultRepo == "" {
					return s, nil
				}

				return fmt.Sprintf("%s/%s", test.defaultRepo, s), nil
			})
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, artifacts)
		})
	}
}
