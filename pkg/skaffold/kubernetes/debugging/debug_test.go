/*
Copyright 2021 The Skaffold Authors

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

package debugging

import (
	"bytes"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// testTransformer is a simple transformer that applies to everything
type testTransformer struct{}

func (t testTransformer) IsApplicable(config debug.ImageConfiguration) bool {
	return true
}

func (t testTransformer) Apply(adapter types.ContainerAdapter, config debug.ImageConfiguration, portAlloc debug.PortAllocator, overrideProtocols []string) (types.ContainerDebugConfiguration, string, error) {
	port := portAlloc(9999)
	container := adapter.GetContainer()
	container.Ports = append(container.Ports, types.ContainerPort{Name: "test", ContainerPort: port})

	testEnv := types.ContainerEnv{Order: []string{"KEY"}, Env: map[string]string{"KEY": "value"}}
	container.Env = testEnv

	return types.ContainerDebugConfiguration{Runtime: "test"}, "", nil
}

func TestPodEncodeDecode(t *testing.T) {
	pod := &v1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "podname"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "name1", Image: "image1"}}}}
	b, err := encodeAsYaml(pod)
	if err != nil {
		t.Errorf("encodeAsYaml() failed: %v", err)
		return
	}
	o, _, err := decodeFromYaml(b, nil, nil)
	if err != nil {
		t.Errorf("decodeFromYaml() failed: %v", err)
		return
	}
	switch o := o.(type) {
	case *v1.Pod:
		testutil.CheckDeepEqual(t, "podname", o.ObjectMeta.Name)
		testutil.CheckDeepEqual(t, 1, len(o.Spec.Containers))
		testutil.CheckDeepEqual(t, "name1", o.Spec.Containers[0].Name)
		testutil.CheckDeepEqual(t, "image1", o.Spec.Containers[0].Image)
	default:
		t.Errorf("decodeFromYaml() failed: expected *v1.Pod but got %T", o)
	}
}

// TestSkipAnnotatedPodSpec verifies that transformPodSpec skips podspecs that have a
// `debug.cloud.google.com/config` annotation.
func TestSkipAnnotatedPodSpec(t *testing.T) {
	tfm := testTransformer{}
	debug.RegisterContainerTransformer(tfm)
	defer debug.UnregisterContainerTransformer(tfm)

	pod := v1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "podname", Annotations: map[string]string{"debug.cloud.google.com/config": "{}"}},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "name1", Image: "image1"}}}}

	retriever := func(image string) (debug.ImageConfiguration, error) {
		return debug.ImageConfiguration{WorkingDir: "/a/dir"}, nil
	}

	copy := pod
	result := transformManifest(&pod, retriever, "HELPERS")
	testutil.CheckDeepEqual(t, false, result)
	testutil.CheckDeepEqual(t, copy, pod) // should be unchanged
}

func TestApplyDebuggingTransforms(t *testing.T) {
	tfm := testTransformer{}
	debug.RegisterContainerTransformer(tfm)
	defer debug.UnregisterContainerTransformer(tfm)

	tests := []struct {
		description string
		shouldErr   bool
		in          string
		out         string
	}{
		{
			"Pod", false,
			`apiVersion: v1
kind: Pod
metadata:
  name: pod
spec:
  containers:
  - image: gcr.io/k8s-debug/debug-example:latest
    name: example
`,
			`apiVersion: v1
kind: Pod
metadata:
  annotations:
    debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
  creationTimestamp: null
  name: pod
spec:
  containers:
  - env:
    - name: KEY
      value: value
    image: gcr.io/k8s-debug/debug-example:latest
    name: example
    ports:
    - containerPort: 9999
      name: test
    resources: {}
status: {}`,
		},
		{
			"Deployment", false,
			`apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 10
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: debug-app
  strategy: {}
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
status: {}`,
		},
		{
			"ReplicaSet", false,
			`apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: my-replicaset
spec:
  replicas: 10
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: apps/v1
kind: ReplicaSet
metadata:
  creationTimestamp: null
  name: my-replicaset
spec:
  replicas: 1
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
status:
  replicas: 0`,
		},
		{
			"StatefulSet", false,
			`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-statefulset
spec:
  replicas: 10
  selector:
    matchLabels:
      app: debug-app
  serviceName: service
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: apps/v1
kind: StatefulSet
metadata:
  creationTimestamp: null
  name: my-statefulset
spec:
  replicas: 1
  selector:
    matchLabels:
      app: debug-app
  serviceName: service
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
  updateStrategy: {}
status:
  availableReplicas: 0
  replicas: 0`,
		},
		{
			"DaemonSet", false,
			`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: my-daemonset
spec:
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: apps/v1
kind: DaemonSet
metadata:
  creationTimestamp: null
  name: my-daemonset
spec:
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
  updateStrategy: {}
status:
  currentNumberScheduled: 0
  desiredNumberScheduled: 0
  numberMisscheduled: 0
  numberReady: 0`,
		},
		{
			"Job", false,
			`apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
spec:
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: batch/v1
kind: Job
metadata:
  creationTimestamp: null
  name: my-job
spec:
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
status: {}`,
		},
		{
			"ReplicationController", false,
			`apiVersion: v1
kind: ReplicationController
metadata:
  name: my-rc
spec:
  replicas: 10
  selector:
    app: debug-app
  template:
    metadata:
      name: debug-pod
      labels:
        app: debug-app
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: v1
kind: ReplicationController
metadata:
  creationTimestamp: null
  name: my-rc
spec:
  replicas: 1
  selector:
    app: debug-app
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
status:
  replicas: 0`,
		},
		{
			description: "skip unhandled yamls like crds",
			shouldErr:   false,
			in: `---
apiVersion: openfaas.com/v1alpha2
kind: Function
metadata:
  name: myfunction
  namespace: openfaas-fn
spec:
  name: myfunction
  image: myfunction`,
			out: `---
apiVersion: openfaas.com/v1alpha2
kind: Function
metadata:
  name: myfunction
  namespace: openfaas-fn
spec:
  name: myfunction
  image: myfunction`,
		},
		{
			"multiple objects", false,
			`apiVersion: v1
kind: Pod
metadata:
  name: pod
spec:
  containers:
  - image: gcr.io/k8s-debug/debug-example:latest
    name: example
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 10
  selector:
    matchLabels:
      app: debug-app
  template:
    metadata:
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - image: gcr.io/k8s-debug/debug-example:latest
        name: example
`,
			`apiVersion: v1
kind: Pod
metadata:
  annotations:
    debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
  creationTimestamp: null
  name: pod
spec:
  containers:
  - env:
    - name: KEY
      value: value
    image: gcr.io/k8s-debug/debug-example:latest
    name: example
    ports:
    - containerPort: 9999
      name: test
    resources: {}
status: {}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: debug-app
  strategy: {}
  template:
    metadata:
      annotations:
        debug.cloud.google.com/config: '{"example":{"runtime":"test"}}'
      creationTimestamp: null
      labels:
        app: debug-app
      name: debug-pod
    spec:
      containers:
      - env:
        - name: KEY
          value: value
        image: gcr.io/k8s-debug/debug-example:latest
        name: example
        ports:
        - containerPort: 9999
          name: test
        resources: {}
status: {}`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			retriever := func(image string) (debug.ImageConfiguration, error) {
				return debug.ImageConfiguration{}, nil
			}

			l, err := manifest.Load(bytes.NewReader([]byte(test.in)))
			t.CheckError(false, err)
			result, err := applyDebuggingTransforms(l, retriever, "HELPERS")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.out, result.String())
		})
	}
}

func TestWorkingDir(t *testing.T) {
	tfm := testTransformer{}
	debug.RegisterContainerTransformer(tfm)
	defer debug.UnregisterContainerTransformer(tfm)

	pod := &v1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "podname"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "name1", Image: "image1"}}}}

	retriever := func(image string) (debug.ImageConfiguration, error) {
		return debug.ImageConfiguration{WorkingDir: "/a/dir"}, nil
	}

	result := transformManifest(pod, retriever, "HELPERS")
	testutil.CheckDeepEqual(t, true, result)
	debugConfig := pod.ObjectMeta.Annotations["debug.cloud.google.com/config"]
	testutil.CheckDeepEqual(t, true, strings.Contains(debugConfig, `"workingDir":"/a/dir"`))
}

func TestArtifactImage(t *testing.T) {
	tfm := testTransformer{}
	debug.RegisterContainerTransformer(tfm)
	defer debug.UnregisterContainerTransformer(tfm)

	pod := &v1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "podname"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "name1", Image: "image1"}}}}

	retriever := func(image string) (debug.ImageConfiguration, error) {
		return debug.ImageConfiguration{Artifact: "gcr.io/random/image"}, nil
	}

	result := transformManifest(pod, retriever, "HELPERS")
	testutil.CheckDeepEqual(t, true, result)
	debugConfig := pod.ObjectMeta.Annotations["debug.cloud.google.com/config"]
	testutil.CheckDeepEqual(t, true, strings.Contains(debugConfig, `"artifact":"gcr.io/random/image"`))
}
