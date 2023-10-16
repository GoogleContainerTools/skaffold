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

func TestSetGKEARMToleration(t *testing.T) {
	tests := []struct {
		description string
		manifest    string
		platforms   PodPlatforms
		expected    string
	}{
		{
			description: "ARM architecture image; no existing tolerations",
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
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - effect: NoSchedule
    key: kubernetes.io/arch
    operator: Equal
    value: arm64`,
		},
		{
			description: "ARM architecture image; existing tolerations",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - key: foo
    operator: Equal
    value: bar
    effect: NoSchedule`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - effect: NoSchedule
    key: foo
    operator: Equal
    value: bar
  - effect: NoSchedule
    key: kubernetes.io/arch
    operator: Equal
    value: arm64`,
		},
		{
			description: "ARM architecture image; existing ARM toleration",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - key: foo
    operator: Equal
    value: bar
    effect: NoSchedule
  - key: kubernetes.io/arch
    operator: Equal
    value: arm64
    effect: NoSchedule`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}, {Architecture: "arm64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - effect: NoSchedule
    key: foo
    operator: Equal
    value: bar
  - effect: NoSchedule
    key: kubernetes.io/arch
    operator: Equal
    value: arm64`,
		},
		{
			description: "non-ARM architecture image; no existing tolerations",
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
				".spec": {{Architecture: "amd64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example`,
		},
		{
			description: "non-ARM architecture image; existing tolerations",
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - key: foo
    operator: Equal
    value: bar
    effect: NoSchedule`,
			platforms: map[string][]spec.Platform{
				".spec": {{Architecture: "amd64", OS: "linux"}},
			},
			expected: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:latest
    name: example
  tolerations:
  - effect: NoSchedule
    key: foo
    operator: Equal
    value: bar`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ml := ManifestList{[]byte(test.manifest)}
			m, err := ml.SetGKEARMToleration(context.Background(), NewResourceSelectorPodSpec(TransformAllowlist, TransformDenylist), test.platforms)
			t.CheckNoError(err)
			expected := ManifestList{[]byte(test.expected)}
			t.CheckDeepEqual(expected.String(), m.String(), testutil.YamlObj(t.T))
		})
	}
}
