/*
Copyright 2018 Google LLC

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

package docker

import (
	"fmt"
	"testing"

	testutil "github.com/GoogleCloudPlatform/skaffold/test"
)

type testImageAPI struct {
	description  string
	imageName    string
	imageID      string
	tagToImageID map[string]string
	shouldErr    bool
	expected     string

	testOpts *testutil.FakeImageAPIOptions
}

func TestRunBuild(t *testing.T) {
	var tests = []testImageAPI{
		{
			description:  "build",
			tagToImageID: map[string]string{},
			imageID:      "sha256:test",
			expected:     "test",
		},
		{
			description:  "bad image build",
			tagToImageID: map[string]string{},
			testOpts: &testutil.FakeImageAPIOptions{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description:  "bad return reader",
			tagToImageID: map[string]string{},
			testOpts: &testutil.FakeImageAPIOptions{
				ReturnBody: &testutil.FakeReaderCloser{Err: fmt.Errorf("")},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			api := testutil.NewFakeImageAPIClient(test.tagToImageID, test.testOpts)
			err := RunBuild(api, &BuildOptions{
				Dockerfile: "Dockerfile",
				ContextDir: ".",
				ImageName:  "finalimage",
			})
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestDigest(t *testing.T) {
	var tests = []testImageAPI{
		{
			description: "get digest",
			imageName:   "identifier",
			tagToImageID: map[string]string{
				"identifier:latest": "sha256:123abc",
			},
			expected: "sha256:123abc",
		},
		{
			description: "image list error",
			imageName:   "test",
			tagToImageID: map[string]string{
				"test:latest": "sha256:123abc",
			},
			testOpts: &testutil.FakeImageAPIOptions{
				ErrImageList: true,
			},
			shouldErr: true,
		},
		{
			description: "not found",
			imageName:   "somethingelse",
			tagToImageID: map[string]string{
				"test:latest": "sha256:123abc",
			},
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			api := testutil.NewFakeImageAPIClient(test.tagToImageID, test.testOpts)
			digest, err := Digest(api, test.imageName)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, digest)
		})
	}
}
