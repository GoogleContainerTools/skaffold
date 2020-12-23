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

package build

import (
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDefaultInitializerGenerateManifests(t *testing.T) {
	tests := []struct {
		description       string
		generatedInfos    []GeneratedArtifactInfo
		mockPorts         []int
		expectedManifests []string
		force             bool
		shouldErr         bool
	}{
		{
			description: "one manifest, force",
			generatedInfos: []GeneratedArtifactInfo{
				{
					ArtifactInfo{
						ImageName: "image1",
					},
					"path/to/manifest",
				},
			},
			expectedManifests: []string{
				`apiVersion: v1
kind: Service
metadata:
  name: image1
  labels:
    app: image1
spec:
  ports:
  - port: 8080
    protocol: TCP
  clusterIP: None
  selector:
    app: image1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image1
  labels:
    app: image1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: image1
  template:
    metadata:
      labels:
        app: image1
    spec:
      containers:
      - name: image1
        image: image1
`,
			},
			force: true,
		},
		{
			description: "2 manifests, 1 port 0, interactive",
			generatedInfos: []GeneratedArtifactInfo{
				{
					ArtifactInfo{
						ImageName: "image1",
					},
					"path/to/manifest1",
				},
				{
					ArtifactInfo{
						ImageName: "image2",
					},
					"path/to/manifest2",
				},
			},
			mockPorts: []int{8080, 0},
			expectedManifests: []string{
				`apiVersion: v1
kind: Service
metadata:
  name: image1
  labels:
    app: image1
spec:
  ports:
  - port: 8080
    protocol: TCP
  clusterIP: None
  selector:
    app: image1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image1
  labels:
    app: image1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: image1
  template:
    metadata:
      labels:
        app: image1
    spec:
      containers:
      - name: image1
        image: image1
`,
				`apiVersion: v1
kind: Service
metadata:
  name: image2
  labels:
    app: image2
spec:
  clusterIP: None
  selector:
    app: image2
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image2
  labels:
    app: image2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: image2
  template:
    metadata:
      labels:
        app: image2
    spec:
      containers:
      - name: image2
        image: image2
`,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			mockPortIdx := 0
			t.Override(&prompt.PortForwardResourceFunc, func(_ io.Writer, imageName string) (int, error) {
				port := test.mockPorts[mockPortIdx]
				mockPortIdx++

				return port, nil
			})

			d := defaultBuildInitializer{
				generatedArtifactInfos: test.generatedInfos,
			}

			manifests, err := d.GenerateManifests(nil, test.force)

			expected := make(map[GeneratedArtifactInfo][]byte)
			for i, info := range test.generatedInfos {
				expected[info] = []byte(test.expectedManifests[i])
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, expected, manifests)
		})
	}
}
