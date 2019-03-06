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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseReference(t *testing.T) {
	var tests = []struct {
		description string
		image       string
		shouldErr   bool
		name        string
		identifier  string
	}{
		{
			description: "simple name",
			image:       "busybox",
			shouldErr:   false,
			name:        "index.docker.io/library/busybox",
			identifier:  "latest",
		},
		{
			description: "with tag",
			image:       "busybox:1.30",
			shouldErr:   false,
			name:        "index.docker.io/library/busybox",
			identifier:  "1.30",
		},
		{
			description: "with tag and digest",
			image:       "ubuntu:latest@sha256:868fd30a0e47b8d8ac485df174795b5e2fe8a6c8f056cc707b232d65b8a1ab68",
			shouldErr:   false,
			name:        "index.docker.io/library/ubuntu",
			identifier:  "sha256:868fd30a0e47b8d8ac485df174795b5e2fe8a6c8f056cc707b232d65b8a1ab68",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ref, err := parseReference(test.image)
			testutil.CheckError(t, test.shouldErr, err)
			testutil.CheckDeepEqual(t, test.name, ref.Context().Name())
			testutil.CheckDeepEqual(t, test.identifier, ref.Identifier())
		})
	}

}
