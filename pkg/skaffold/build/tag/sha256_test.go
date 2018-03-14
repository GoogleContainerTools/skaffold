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

	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

func TestGenerateFullyQualifiedImageName(t *testing.T) {
	var tests = []struct {
		description string
		opts        *TagOptions
		digest      string
		image       string
		expected    string

		shouldErr bool
	}{
		{
			description: "no error",
			opts: &TagOptions{
				ImageName: "test",
				Digest:    "sha256:12345abcde",
			},
			expected: "test:12345abcde",
		},
		{
			description: "wrong digest format",
			opts: &TagOptions{
				ImageName: "test",
				Digest:    "wrong:digest:format",
			},
			shouldErr: true,
		},
		{
			description: "wrong digest format no colon",
			opts: &TagOptions{
				ImageName: "test",
				Digest:    "sha256",
			},
			shouldErr: true,
		},
		{
			description: "error no tag opts",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &ChecksumTagger{}
			tag, err := c.GenerateFullyQualifiedImageName(".", test.opts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, tag)
		})
	}
}
