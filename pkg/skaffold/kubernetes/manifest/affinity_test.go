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

package manifest

import (
	"context"
	"testing"

	spec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetImagePlatforms(t *testing.T) {
	tests := []struct {
		description string
		manifest    string
		platforms   PodPlatforms
		expected    PodPlatforms
	}{
		{
			description: "single container in Pod",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
			platforms: map[string][]spec.Platform{
				"gcr.io/k8s-skaffold/example:latest": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
		},
		{
			description: "multiple containers in Pod with overlapping platforms",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
			platforms: map[string][]spec.Platform{
				"gcr.io/k8s-skaffold/example1:latest": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
				"gcr.io/k8s-skaffold/example2:latest": {{Architecture: "arm64", OS: "linux"}, {Architecture: "arm", OS: "freebsd"}},
			},
			expected: map[string][]spec.Platform{
				".spec": {{Architecture: "arm64", OS: "linux"}},
			},
		},
		{
			description: "multiple containers in Pod with no overlapping platforms",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
			platforms: map[string][]spec.Platform{
				"gcr.io/k8s-skaffold/example1:latest": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
				"gcr.io/k8s-skaffold/example2:latest": {{Architecture: "arm", OS: "freebsd"}},
			},
			expected: map[string][]spec.Platform{
				".spec": nil,
			},
		},
		{
			description: "deployment manifest",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: getting-started
  labels:
    app: getting-started
spec:
  replicas: 3
  selector:
    matchLabels:
      app: getting-started
  template:
    metadata:
      labels:
        app: getting-started
    spec:
      containers:
      - name: example
        image: gcr.io/k8s-skaffold/example:latest`,
			platforms: map[string][]spec.Platform{
				"gcr.io/k8s-skaffold/example:latest": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: map[string][]spec.Platform{
				".spec.template.spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getPlatforms, func(image string) ([]spec.Platform, error) {
				return test.platforms[image], nil
			})
			ml := ManifestList{[]byte(test.manifest)}
			pl, err := ml.GetImagePlatforms(context.Background(), NewResourceSelectorImages(TransformAllowlist, TransformDenylist))
			t.CheckNoError(err)
			t.CheckMapsMatch(test.expected, pl)
		})
	}
}

func TestSetPlatformNodeAffinity(t *testing.T) {
	tests := []struct {
		description string
		manifest    string
		platforms   PodPlatforms
		expected    string
	}{
		{
			description: "single container in Pod",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - amd64
        - matchExpressions:
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - arm64
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
		},
		{
			description: "multiple containers in Pod with overlapping platforms",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - arm64
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
		},
		{
			description: "multiple containers in Pod with no overlapping platforms",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
			platforms: map[string][]spec.Platform{
				".spec": nil,
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example1:latest
    name: example1
  - image: gcr.io/k8s-skaffold/example2:latest
    name: example2`,
		},
		{
			description: "deployment manifest",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: getting-started
  labels:
    app: getting-started
spec:
  replicas: 3
  selector:
    matchLabels:
      app: getting-started
  template:
    metadata:
      labels:
        app: getting-started
    spec:
      containers:
      - name: example
        image: gcr.io/k8s-skaffold/example:latest`,
			platforms: map[string][]spec.Platform{
				".spec.template.spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: getting-started
  name: getting-started
spec:
  replicas: 3
  selector:
    matchLabels:
      app: getting-started
  template:
    metadata:
      labels:
        app: getting-started
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values:
                - linux
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64
            - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values:
                - linux
              - key: kubernetes.io/arch
                operator: In
                values:
                - arm64
      containers:
      - image: gcr.io/k8s-skaffold/example:latest
        name: example`,
		},
		{
			description: "existing node affinity",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: foo1
            operator: Equals
            values:
            - bar1
        - matchExpressions:
          - key: foo2
            operator: Equals
            values:
            - bar2
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: foo1
            operator: Equals
            values:
            - bar1
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - amd64
        - matchExpressions:
          - key: foo2
            operator: Equals
            values:
            - bar2
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - amd64
        - matchExpressions:
          - key: foo1
            operator: Equals
            values:
            - bar1
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - arm64
        - matchExpressions:
          - key: foo2
            operator: Equals
            values:
            - bar2
          - key: kubernetes.io/os
            operator: In
            values:
            - linux
          - key: kubernetes.io/arch
            operator: In
            values:
            - arm64
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getPlatforms, func(image string) ([]spec.Platform, error) {
				return test.platforms[image], nil
			})
			ml := ManifestList{[]byte(test.manifest)}
			m, err := ml.SetPlatformNodeAffinity(context.Background(), NewResourceSelectorPodSpec(TransformAllowlist, TransformDenylist), test.platforms)
			t.CheckNoError(err)
			expected := ManifestList{[]byte(test.expected)}
			t.CheckDeepEqual(expected.String(), m.String(), testutil.YamlObj(t.T))
		})
	}
}
