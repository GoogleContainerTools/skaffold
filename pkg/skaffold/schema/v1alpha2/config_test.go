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

package v1alpha2

import (
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

func TestRecursiveGetImages(t *testing.T) {
	var tests = []struct {
		description string
		yaml        string
		expected    []string
	}{
		{
			description: "get one image",
			yaml:        `image: image1`,
			expected:    []string{"image1"},
		},
		{
			description: "get multiple images",
			yaml: `apiVersion: v1
kind: Pod
metadata:
    name: getting-started
spec:
    containers:
    - name: getting-started
    image: image
image: image2`,
			expected: []string{"image", "image2"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m := map[interface{}]interface{}{}
			if err := yaml.Unmarshal([]byte(test.yaml), &m); err != nil {
				t.Fatal(err)
			}
			actual := recursiveGetImages(m)
			sort.Strings(actual)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
