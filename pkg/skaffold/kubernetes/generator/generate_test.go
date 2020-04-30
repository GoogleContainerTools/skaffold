/*
Copyright 2020 The Skaffold Authors

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

package generator

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestManifestGeneration(t *testing.T) {
	tests := []struct {
		description      string
		images           []string
		expectedManifest string
	}{
		{
			description: "single image",
			images:      []string{"foo"},
			expectedManifest: `apiVersion: v1
kind: Service
metadata:
  name: foo
  labels:
    app: foo
spec:
  clusterIP: None
  selector:
    app: foo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  labels:
    app: foo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: foo
  template:
    metadata:
      labels:
        app: foo
    spec:
      containers:
      - name: foo
        image: foo
`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			for _, image := range test.images {
				manifest, err := Generate(image)

				t.CheckNoError(err)
				t.CheckDeepEqual(test.expectedManifest, string(manifest))
			}
		})
	}
}
