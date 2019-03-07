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

package initializer

import (
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsSupportedKubernetesFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "valid k8 yaml filename format",
			filename: "test1.yaml",
			expected: true,
		},
		{
			name:     "valid k8 json filename format",
			filename: "test1.json",
			expected: true,
		},
		{
			name:     "valid k8 yaml filename format",
			filename: "test1.yml",
			expected: true,
		},
		{
			name:     "invalid file",
			filename: "some.config",
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if IsSupportedKubernetesFormat(test.filename) != test.expected {
				t.Errorf("expected to see %t for %s, but instead got %t", test.expected,
					test.filename, !test.expected)
			}
		})
	}
}

func TestIsSkaffoldConfig(t *testing.T) {
	tests := []struct {
		name     string
		contents []byte
		expected bool
	}{
		{
			name: "valid skaffold config",
			contents: []byte(`apiVersion: skaffold/v1beta6
kind: Config
deploy:
  kustomize: {}`),
			expected: true,
		},
		{
			name:     "not a valid format",
			contents: []byte("test"),
			expected: false,
		},
		{
			name: "invalid skaffold config version",
			contents: []byte(`apiVersion: skaffold/v2beta1
kind: Config
deploy:
  kustomize: {}`),
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := testutil.CreateTempFileWithContents(t, "", "skaffold.yaml", test.contents)
			defer os.Remove(filename) // clean up
			if IsSkaffoldConfig(filename) != test.expected {
				t.Errorf("expected to see %t for\n%s. but instead got %t", test.expected,
					test.contents, !test.expected)
			}
		})
	}
}
