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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/spf13/afero"
)

func TestSupportedKubernetesFormats(t *testing.T) {
	var tests = []struct {
		description string
		in          string
		out         bool
	}{
		{
			description: "yaml",
			in:          "filename.yaml",
			out:         true,
		},
		{
			description: "yml",
			in:          "filename.yml",
			out:         true,
		},
		{
			description: "json",
			in:          "filename.json",
			out:         true,
		},
		{
			description: "txt",
			in:          "filename.txt",
			out:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actual := IsSupportedKubernetesFormat(tt.in)
			if tt.out != actual {
				t.Errorf("out: %t, actual: %t", tt.out, actual)
			}
		})
	}
}

func TestExpandPathsGlob(t *testing.T) {
	defer func(fs afero.Fs) { Fs = fs }(Fs)
	Fs = afero.NewMemMapFs()

	Fs.MkdirAll("dir/sub_dir", 0700)
	Fs.MkdirAll("dir_b/sub_dir_b", 0700)
	afero.WriteFile(Fs, "dir_b/sub_dir_b/file", []byte(""), 0650)
	afero.WriteFile(Fs, "dir/sub_dir/file", []byte(""), 0650)

	var tests = []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "match exact filename",
			in:          []string{"dir/sub_dir/file"},
			out:         []string{"dir/sub_dir/file"},
		},
		{
			description: "match leaf directory glob",
			in:          []string{"dir/sub_dir/*"},
			out:         []string{"dir/sub_dir/file"},
		},
		{
			description: "match top level glob",
			in:          []string{"dir*"},
			out:         []string{"dir/sub_dir/file", "dir_b/sub_dir_b/file"},
		},
		{
			description: "error unmatched glob",
			in:          []string{"dir/sub_dir_c/*"},
			shouldErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actual, err := ExpandPathsGlob(tt.in)
			testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.out, actual)
		})
	}
}

func TestSplitTag(t *testing.T) {
	var tests = []struct {
		description            string
		image                  string
		expectedName           string
		expectedFullyQualified bool
	}{
		{
			description:            "port and tag",
			image:                  "host:1234/user/container:tag",
			expectedName:           "host:1234/user/container",
			expectedFullyQualified: true,
		},
		{
			description:            "port",
			image:                  "host:1234/user/container",
			expectedName:           "host:1234/user/container",
			expectedFullyQualified: false,
		},
		{
			description:            "tag",
			image:                  "host/user/container:tag",
			expectedName:           "host/user/container",
			expectedFullyQualified: true,
		},
		{
			description:            "latest",
			image:                  "host/user/container:latest",
			expectedName:           "host/user/container",
			expectedFullyQualified: false,
		},
		{
			description:            "digest",
			image:                  "gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedName:           "gcr.io/k8s-skaffold/example",
			expectedFullyQualified: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			parsed, err := ParseReference(test.image)

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedName, parsed.BaseName)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFullyQualified, parsed.FullyQualified)
		})
	}
}
