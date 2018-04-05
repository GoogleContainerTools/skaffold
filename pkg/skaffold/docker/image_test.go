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
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

func TestMain(m *testing.M) {
	// So we don't shell out to credentials helpers or try to read dockercfg
	auth := DefaultAuthHelper
	defer func() { DefaultAuthHelper = auth }()
	DefaultAuthHelper = testAuthHelper{}
	os.Exit(m.Run())
}

type testImageAPI struct {
	description  string
	imageName    string
	imageID      string
	tagToImageID map[string]string
	shouldErr    bool
	expected     string

	testOpts *testutil.FakeImageAPIOptions
}

func TestRunPush(t *testing.T) {
	var tests = []testImageAPI{
		{
			description:  "push",
			imageName:    "gcr.io/scratchman",
			tagToImageID: map[string]string{},
		},
		{
			description:  "no error pushing non canonical tag",
			imageName:    "noncanonicalscratchman",
			tagToImageID: map[string]string{},
		},
		{
			description:  "no error pushing canonical tag",
			imageName:    "canonical/name",
			tagToImageID: map[string]string{},
		},
		{
			description:  "stream error",
			imageName:    "gcr.io/imthescratchman",
			tagToImageID: map[string]string{},
			testOpts: &testutil.FakeImageAPIOptions{
				ReturnBody: &testutil.FakeReaderCloser{Err: fmt.Errorf("")},
			},
			shouldErr: true,
		},
		{
			description:  "image push error",
			imageName:    "gcr.io/skibabopbadopbop",
			tagToImageID: map[string]string{},
			testOpts: &testutil.FakeImageAPIOptions{
				ErrImagePush: true,
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			api := testutil.NewFakeImageAPIClient(test.tagToImageID, test.testOpts)
			err := RunPush(context.Background(), api, test.imageName, &bytes.Buffer{})
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
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
			err := RunBuild(context.Background(), api, &BuildOptions{
				Dockerfile: "Dockerfile",
				ContextDir: "../../../testdata/docker",
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
			imageName:   "identifier:latest",
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
			digest, err := Digest(context.Background(), api, test.imageName)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, digest)
		})
	}
}
