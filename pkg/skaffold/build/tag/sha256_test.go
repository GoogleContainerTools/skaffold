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

package tag

import (
	"testing"

	testutil "github.com/GoogleCloudPlatform/skaffold/test"
)

func TestNewChecksumTaggerFromDigest(t *testing.T) {
	var tests = []struct {
		description string
		digest      string
		image       string

		shouldErr bool
	}{
		{
			description: "no error",
			digest:      "sha256:12345abcde",
			image:       "test",
		},
		{
			description: "wrong digest format",
			digest:      "wrong:digest:format",
			image:       "test",
			shouldErr:   true,
		},
		{
			description: "wrong digest format no colon",
			digest:      "sha256",
			image:       "test",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			_, err := NewChecksumTaggerFromDigest(test.digest, test.image)
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestGenerateFullyQualifiedImageName(t *testing.T) {
	tagger := &ChecksumTagger{ImageName: "test", Checksum: "1234"}
	if _, err := tagger.GenerateFullyQualifiedImageName(); err != nil {
		t.Errorf("Error generating tag: %s", err)
	}
}
