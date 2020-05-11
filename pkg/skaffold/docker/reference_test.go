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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		description            string
		image                  string
		expectedName           string
		expectedTag            string
		expectedDigest         string
		expectedFullyQualified bool
		shouldErr              bool
	}{
		{
			description:            "port and tag",
			image:                  "host:1234/user/container:tag",
			expectedName:           "host:1234/user/container",
			expectedTag:            "tag",
			expectedFullyQualified: true,
		},
		{
			description:            "port",
			image:                  "host:1234/user/container",
			expectedName:           "host:1234/user/container",
			expectedTag:            "",
			expectedFullyQualified: false,
		},
		{
			description:            "tag",
			image:                  "host/user/container:tag",
			expectedName:           "host/user/container",
			expectedTag:            "tag",
			expectedFullyQualified: true,
		},
		{
			description:            "latest",
			image:                  "host/user/container:latest",
			expectedName:           "host/user/container",
			expectedTag:            "latest",
			expectedFullyQualified: false,
		},
		{
			description:            "digest",
			image:                  "gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedName:           "gcr.io/k8s-skaffold/example",
			expectedDigest:         "sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedFullyQualified: true,
		},
		{
			description:            "digest and tag",
			image:                  "gcr.io/k8s-skaffold/example:v1@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedName:           "gcr.io/k8s-skaffold/example",
			expectedTag:            "v1",
			expectedDigest:         "sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedFullyQualified: true,
		},
		{
			description:            "docker library",
			image:                  "nginx:latest",
			expectedName:           "nginx",
			expectedTag:            "latest",
			expectedFullyQualified: false,
		},
		{
			description: "invalid reference",
			image:       "!!invalid!!",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			parsed, err := ParseReference(test.image)

			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedName, parsed.BaseName)
				t.CheckDeepEqual(test.expectedTag, parsed.Tag)
				t.CheckDeepEqual(test.expectedDigest, parsed.Digest)
				t.CheckDeepEqual(test.expectedFullyQualified, parsed.FullyQualified)
			}
		})
	}
}
