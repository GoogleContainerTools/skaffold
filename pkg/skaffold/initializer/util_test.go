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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsSupportedKubernetesFileExtension(t *testing.T) {
	tests := []struct {
		description string
		filename    string
		expected    bool
	}{
		{
			description: "valid k8 yaml filename format",
			filename:    "test1.yaml",
			expected:    true,
		},
		{
			description: "valid k8 json filename format",
			filename:    "test1.json",
			expected:    true,
		},
		{
			description: "valid k8 yaml filename format",
			filename:    "test1.yml",
			expected:    true,
		},
		{
			description: "invalid file",
			filename:    "some.config",
			expected:    false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			supported := IsSupportedKubernetesFileExtension(test.filename)

			t.CheckDeepEqual(test.expected, supported)
		})
	}
}

func TestIsSkaffoldConfig(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		isValid     bool
	}{
		{
			description: "valid skaffold config",
			contents: `apiVersion: skaffold/v1beta6
kind: Config
deploy:
  kustomize: {}`,
			isValid: true,
		},
		{
			description: "not a valid format",
			contents:    "test",
			isValid:     false,
		},
		{
			description: "invalid skaffold config version",
			contents: `apiVersion: skaffold/v2beta1
kind: Config
deploy:
  kustomize: {}`,
			isValid: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("skaffold.yaml", test.contents)

			isValid := IsSkaffoldConfig(tmpDir.Path("skaffold.yaml"))

			t.CheckDeepEqual(test.isValid, isValid)
		})
	}
}
