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

package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestKubectlRenderOutput(t *testing.T) {
	ns, _ := SetupNamespace(t)
	test := struct {
		description string
		builds      []graph.Artifact
		input       string
		expectedOut string
	}{
		description: "write rendered manifest to provided filepath",
		builds: []graph.Artifact{
			{
				ImageName: "us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold",
				Tag:       "us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold:test",
			},
		},
		input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold
    name: skaffold
`,
		expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  namespace: %s
spec:
  containers:
  - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold:test
    name: skaffold`, ns.Name)}

	testutil.Run(t, test.description, func(t *testutil.T) {
		MarkIntegrationTest(t.T, CanRunWithoutGcp)
		tmpDir := t.NewTempDir()
		tmpDir.Write("deployment.yaml", test.input).Chdir()

		rc := latest.RenderConfig{
			Generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"}},
		}
		mockCfg := render.MockConfig{WorkingDir: tmpDir.Root()}
		r, err := kubectl.New(mockCfg, rc, map[string]string{}, "default", ns.Name, nil, true)
		t.RequireNoError(err)
		var b bytes.Buffer
		l, err := r.Render(context.Background(), &b, test.builds, false)

		t.CheckNoError(err)

		t.CheckDeepEqual(test.expectedOut, l.String(), testutil.YamlObj(t.T))
	})
}

func TestKubectlRender(t *testing.T) {
	ns, _ := SetupNamespace(t)
	tests := []struct {
		description string
		builds      []graph.Artifact
		input       string
		expectedOut string
	}{
		{
			description: "normal render",
			builds: []graph.Artifact{
				{
					ImageName: "us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold",
					Tag:       "us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold:test",
				},
			},
			input: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold
    name: skaffold
`,
			expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: %s
spec:
  containers:
  - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold:test
    name: skaffold`, ns.Name),
		},
		{
			description: "two artifacts",
			builds: []graph.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			input: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: %s
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2`, ns.Name),
		},
		{
			description: "two artifacts, combined manifests",
			builds: []graph.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			input: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-456
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: %s
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-456
  namespace: %s
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2`, ns.Name, ns.Name),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			tmpDir := t.NewTempDir()
			tmpDir.Write("deployment.yaml", test.input).
				Chdir()
			rc := latest.RenderConfig{
				Generate: latest.Generate{
					RawK8s: []string{"deployment.yaml"}},
			}
			mockCfg := render.MockConfig{WorkingDir: tmpDir.Root()}
			r, err := kubectl.New(mockCfg, rc, map[string]string{}, "default", ns.Name, nil, true)
			t.RequireNoError(err)
			var b bytes.Buffer
			l, err := r.Render(context.Background(), &b, test.builds, false)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, l.String(), testutil.YamlObj(t.T))
		})
	}
}

func TestHelmRender(t *testing.T) {
	tests := []struct {
		description      string
		dir              string
		args             []string
		builds           []graph.Artifact
		helmReleases     []latest.HelmRelease
		expectedOut      string
		withoutBuildJSON bool
	}{
		{
			description: "Bare bones render",
			dir:         "testdata/gke_loadbalancer-render",
			expectedOut: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: gke-loadbalancer
    skaffold.dev/run-id: phony-run-id
  name: gke-loadbalancer
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    app: gke-loadbalancer
  type: LoadBalancer
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: gke-loadbalancer
    skaffold.dev/run-id: phony-run-id
  name: gke-loadbalancer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gke-loadbalancer
  template:
    metadata:
      labels:
        app: gke-loadbalancer
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: gke-loadbalancer:test
        name: gke-container
        ports:
        - containerPort: 3000
`,
		},
		{
			description: "A more complex template",
			dir:         "testdata/helm-render",
			args:        []string{"--profile=helm-render"},
			expectedOut: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm-skaffold-helm
spec:
  ports:
  - name: nginx
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: skaffold-helm
    release: skaffold-helm
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: skaffold-helm
      release: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        release: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold-helm:sha256-nonsenselettersandnumbers
        imagePullPolicy: always
        name: skaffold-helm
        ports:
        - containerPort: 80
        resources: {}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations: null
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
  name: skaffold-helm-skaffold-helm
spec:
  rules:
  - http:
      paths:
      - backend:
          service:
            name: skaffold-helm-skaffold-helm
            port:
              number: 80
        path: /
        pathType: ImplementationSpecific
`,
		}, {
			description:      "Template with Release.namespace set from skaffold.yaml file deploy.helm.releases.namespace",
			dir:              "testdata/helm-deploy-namespace",
			withoutBuildJSON: true,
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace
spec:
  replicas: 2
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
`,
		}, {
			description:      "Template with Release.namespace set from skaffold.yaml file manifests.helm.releases.namespace",
			dir:              "testdata/helm-manifests-namespace",
			withoutBuildJSON: true,
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace
spec:
  replicas: 2
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
`,
		}, {
			description:      "Template with Release.namespace set from skaffold.yaml file manifests.helm.releases.namespace and deploy.helm.releases.namespace",
			dir:              "testdata/helm-manifests-and-deploy-namespace",
			withoutBuildJSON: true,
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace-1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace-2
spec:
  replicas: 2
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
`,
		}, {
			description:      "Template with Release.namespace set from skaffold.yaml file deploy.helm.releases.namespace - v1 skaffold schema",
			dir:              "testdata/helm-deploy-namespace-v1-schema",
			withoutBuildJSON: true,
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace
spec:
  replicas: 2
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
`,
		}, {
			description:      "Template replicaCount with --set flag",
			dir:              "testdata/helm-render-simple",
			args:             []string{"--set", "replicaCount=3"},
			withoutBuildJSON: true,
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
  namespace: helm-namespace
spec:
  replicas: 3
  selector:
    matchLabels:
      app: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: skaffold-helm:latest
        name: skaffold-helm
`,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			args := []string{"--default-repo=", "--label=skaffold.dev/run-id=phony-run-id"}
			if !test.withoutBuildJSON {
				args = append(args, "--build-artifacts=builds.out.json")
			}
			args = append(args, test.args...)

			out := skaffold.Render(args...).InDir(test.dir).RunOrFailOutput(t)

			testutil.CheckDeepEqual(t, test.expectedOut, string(out), testutil.YamlObj(t))
		})
	}
}

func TestRenderWithBuilds(t *testing.T) {
	tests := []struct {
		description         string
		config              string
		buildOutputFilePath string
		images              string
		offline             bool
		input               map[string]string // file path => content
		expectedOut         string
		namespaceFlag       string
	}{
		{
			description: "kubectl render from build output, online, no labels",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kubectl:
    manifests:
      - deployment.yaml
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             false,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`},
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
		{
			description: "kubectl render from build output, offline, no labels",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kubectl:
    manifests:
      - deployment.yaml
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             true,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`},
			// No `metadata.namespace` in offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},

		{
			description: "kubectl render from build output, offline, with labels",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kubectl:
    manifests:
      - deployment.yaml
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             true,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`},
			// No `metadata.namespace` in offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},

		{
			description: "kustomize render from build output, offline, no labels",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kustomize: {} # defaults to current working directory
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             true,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"kustomization.yaml": `
commonLabels:
  this-is-from: kustomization.yaml

resources:
  - deployment.yaml
`},
			// No `metadata.namespace` in offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    this-is-from: kustomization.yaml
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},

		{
			description: "kustomize render from build output, offline, with labels",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kustomize: {} # defaults to current working directory
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             true,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"kustomization.yaml": `
commonLabels:
  this-is-from: kustomization.yaml

resources:
  - deployment.yaml
`},
			// No `metadata.namespace` in offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    this-is-from: kustomization.yaml
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
		{
			description: "kustomize render from build output, offline=false",
			config: `
apiVersion: skaffold/v4beta2
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

manifests:
  kustomize:
    paths:
    - .
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             false,
			input: map[string]string{"pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"kustomization.yaml": `
commonLabels:
  this-is-from: kustomization.yaml

resources:
  - pod.yaml
`},
			// No `metadata.namespace` should be injected
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    this-is-from: kustomization.yaml
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
		{
			description: "kustomize + rawYaml render from build output, offline=false",
			config: `
apiVersion: skaffold/v4beta2
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

manifests:
  rawYaml:
    - pod2.yaml
  kustomize:
    paths:
    - .
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             false,
			input: map[string]string{"pod1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-1
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"pod2.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-2
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"kustomization.yaml": `
commonLabels:
  this-is-from: kustomization.yaml

resources:
  - pod1.yaml
`},
			// No `metadata.namespace` should be injected in the manifest rendererd by kustomize
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-2
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    this-is-from: kustomization.yaml
  name: my-pod-1
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
		{
			description:   "kustomize + rawYaml render from build output, offline=false, with namespace flag",
			namespaceFlag: "mynamespace",
			config: `
apiVersion: skaffold/v4beta2
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

manifests:
  rawYaml:
    - pod2.yaml
  kustomize:
    paths:
    - .
`,
			buildOutputFilePath: "testdata/render/build-output.json",
			offline:             false,
			input: map[string]string{"pod1.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-1
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"pod2.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-2
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`,
				"kustomization.yaml": `
commonLabels:
  this-is-from: kustomization.yaml

resources:
  - pod1.yaml
`},
			// No `metadata.namespace` should be injected in the manifest rendererd by kustomize
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-2
  namespace: mynamespace
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    this-is-from: kustomization.yaml
  name: my-pod-1
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
		{
			description: "kubectl render with images",
			config: `
apiVersion: skaffold/v2alpha1
kind: Config

# Irrelevant for rendering from previous build output
build:
  artifacts: []

deploy:
  kubectl:
    manifests:
      - deployment.yaml
`,
			images:  "12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf,gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc",
			offline: false,
			input: map[string]string{"deployment.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a
    name: a
  - image: gcr.io/my/project-b
    name: b
`},
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: 12345.dkr.ecr.eu-central-1.amazonaws.com/my/project-a:4da6a56988057d23f68a4e988f4905dd930ea438-dirty@sha256:d8a33c260c50385ea54077bc7032dba0a860dc8870464f6795fd0aa548d117bf
    name: a
  - image: gcr.io/my/project-b:764841f8bac17e625724adcbf0d28013f22d058f-dirty@sha256:79e160161fd8190acae2d04d8f296a27a562c8a59732c64ac71c99009a6e89bc
    name: b
`,
		},
	}

	testDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			tmpDir := t.NewTempDir()
			tmpDir.Write("skaffold.yaml", test.config)

			for filePath, content := range test.input {
				tmpDir.Write(filePath, content)
			}

			tmpDir.Chdir()

			// `--default-repo=` is used to cancel the default repo that is set by default.
			args := []string{"--default-repo=", "--digest-source=local", "--output", "rendered.yaml"}
			if test.buildOutputFilePath != "" {
				args = append(args, "--build-artifacts="+path.Join(testDir, test.buildOutputFilePath))
			} else {
				args = append(args, "--images="+test.images)
			}

			if test.namespaceFlag != "" {
				args = append(args, "--namespace", test.namespaceFlag)
			}

			if test.offline {
				env := []string{"KUBECONFIG=not-supposed-to-be-used-in-offline-mode"}
				args = append(args, "--offline=true")
				skaffold.Render(args...).WithEnv(env).RunOrFail(t.T)
			} else {
				skaffold.Render(args...).RunOrFail(t.T)
			}

			fileContent, err := os.ReadFile("rendered.yaml")
			t.RequireNoError(err)

			// Tests are written in a way that actual output is valid YAML
			parsed := make(map[string]interface{})
			err = yaml.UnmarshalStrict(fileContent, parsed)
			t.CheckNoError(err)

			fileContentReplaced := regexp.MustCompile("(?m)(skaffold.dev/run-id|skaffold.dev/docker-api-version): .+$").ReplaceAll(fileContent, []byte("$1: SOMEDYNAMICVALUE"))

			t.RequireNoError(err)
			t.CheckDeepEqual(test.expectedOut, string(fileContentReplaced), testutil.YamlObj(t.T))
		})
	}
}

func TestRenderHydrationDirCreation(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	const hydrationDir = "hydration-dir"

	tests := []struct {
		description              string
		shouldCreateHydrationDir bool
		config                   string
		manifest                 string
	}{
		{
			description:              "project with kpt renderer should create hydration dir",
			shouldCreateHydrationDir: true,
			config: `
apiVersion: skaffold/v4beta1
kind: Config

build:
  artifacts: []
manifests:
  kpt: []
`,
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
		},
		{
			description:              "project with kpt deployer should create hydration dir",
			shouldCreateHydrationDir: true,
			config: `
apiVersion: skaffold/v4beta1
kind: Config

build:
  artifacts: []
manifests:
  kpt: []
deploy:
  kpt: {}
`,
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
		},
		{
			description:              "project with rawYaml and transform should not create hydration dir (uses kpt renderer)",
			shouldCreateHydrationDir: false,
			config: `
apiVersion: skaffold/v4beta1
kind: Config

build:
  artifacts: []
manifests:
  rawYaml:
    - k8s/k8s-pod.yaml
  transform:
    - name: set-annotations
      configMap:
        - "author:fake-author"
`,
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
		},
		{
			description:              "project without kpt should not create hydration dir",
			shouldCreateHydrationDir: false,
			config: `
apiVersion: skaffold/v4beta1
kind: Config

build:
  artifacts: []
manifests:
  rawYaml:
    - k8s/k8s-pod.yaml
`,
			manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - name: getting-started
    image: skaffold-example`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			tmpDir.Mkdir("k8s")
			tmpDir.Write("skaffold.yaml", test.config)
			tmpDir.Write("k8s/k8s-pod.yaml", test.manifest)
			tmpDir.Chdir()

			args := []string{"--hydration-dir", hydrationDir}

			skaffold.Render(args...).RunOrFail(t.T)

			_, err := os.Stat(filepath.Join(tmpDir.Root(), hydrationDir))

			if test.shouldCreateHydrationDir {
				t.CheckFalse(os.IsNotExist(err))
			} else {
				t.CheckTrue(os.IsNotExist(err))
			}
		})
	}
}

func TestRenderParameterization(t *testing.T) {
	tests := []struct {
		description        string
		args               []string
		WithBuildArtifacts bool
		config             string
		input              map[string]string // file path => content
		expectedOut        string
	}{
		{
			description: "kubectl set manifest label with apply-setters",
			args:        []string{"--offline=true"},
			config: `apiVersion: skaffold/v4beta2
kind: Config
manifests:
  rawYaml:
    - k8s-pod.yaml
  transform:
    - name: apply-setters
      configMap:
        - "app1:from-apply-setters-1"
`,
			input: map[string]string{
				"k8s-pod.yaml": `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: hhhh # kpt-set: ${app1}
spec:
  containers:
  - name: getting-started
    image: skaffold-example`,
			}, expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: from-apply-setters-1
spec:
  containers:
  - name: getting-started
    image: skaffold-example
`,
		},
		{description: "kustomize set annotation with set-annotations transformer",
			args: []string{"--offline=true"},
			config: `apiVersion: skaffold/v4beta2
kind: Config
manifests:
  kustomize:
    paths:
      - .
  transform:
    - name: set-annotations
      configMap:
        - "author:fake-author"`,
			input: map[string]string{
				"kustomization.yaml": `resources:
  - deployment.yaml
patchesStrategicMerge:
  - patch.yaml
`, "deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-test
  labels:
    app: kustomize-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kustomize-test
  template:
    metadata:
      labels:
        app: kustomize-test
    spec:
      containers:
        - name: kustomize-test
          image: not/a/valid/image
`, "patch.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kustomize-test
spec:
  template:
    spec:
      containers:
        - name: kustomize-test
          image: index.docker.io/library/busybox
          command:
            - sleep
            - "3600"
`},
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    author: fake-author
  labels:
    app: kustomize-test
  name: kustomize-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kustomize-test
  template:
    metadata:
      annotations:
        author: fake-author
      labels:
        app: kustomize-test
    spec:
      containers:
        - command:
            - sleep
            - "3600"
          image: index.docker.io/library/busybox
          name: kustomize-test
`},
		{
			description: "kustomize/overlay parameterization with --set flag",
			args:        []string{"--offline", "--set", "env2=222a", "--set", "app1=111a"},
			config: `apiVersion: skaffold/v4beta2
kind: Config
metadata:
  name: getting-started-kustomize
manifests:
  kustomize:
    paths:
    - overlays/dev
`, input: map[string]string{"base/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
`, "base/deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-kustomize
  labels:
    app: skaffold-kustomize # from-param: ${app1}
spec:
  selector:
    matchLabels:
      app: skaffold-kustomize
  template:
    metadata:
      labels:
        app: skaffold-kustomize
    spec:
      containers:
      - name: skaffold-kustomize
        image: skaffold-kustomize
`, "overlays/dev/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# namespace: dev
nameSuffix: -dev

patchesStrategicMerge:
- deployment.yaml

resources:
- ../../base
`, "overlays/dev/deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-kustomize
  labels:
    env: dev # from-param: ${env2}
`}, expectedOut: `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: 111a
    env: 222a
  name: skaffold-kustomize-dev
spec:
  selector:
    matchLabels:
      app: skaffold-kustomize
  template:
    metadata:
      labels:
        app: skaffold-kustomize
    spec:
      containers:
      - image: skaffold-kustomize
        name: skaffold-kustomize
`,
		},
		{description: "test set transformer values with value file",
			args: []string{"--offline=true", "--set-value-file", "values.env"},
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  rawYaml:
    - k8s-pod.yaml
`,
			input: map[string]string{
				"k8s-pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: hhhh # from-param: ${app1}
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
				"values.env": `app1=from-file`,
			}, expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: from-file
spec:
  containers:
    - name: getting-started
      image: skaffold-example
`,
		},
		{
			description: "test set transformer values with value file and value from command-line",
			args:        []string{"--offline=true", "--set-value-file", "values.env", "--set", "app2=from-command-line"},
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  rawYaml:
    - k8s-pod.yaml
  transform:
    - name: apply-setters
      configMap:
        - "app1:from-apply-setters-1"
`,
			input: map[string]string{
				"k8s-pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: hhhh # from-param: ${app1}
    b: hhhh # from-param: ${app2}
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
				"values.env": `app1=from-file`,
			}, expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: from-file
    b: from-command-line
spec:
  containers:
    - name: getting-started
      image: skaffold-example
`},
		{
			description: "test set transformer file values respect values from command-line",
			args:        []string{"--offline=true", "--set-value-file", "values.env", "--set", "app1=from-command-line"},
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  rawYaml:
    - k8s-pod.yaml
  transform:
    - name: apply-setters
      configMap:
        - "app1:from-apply-setters-1"
`,
			input: map[string]string{
				"k8s-pod.yaml": `
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: hhhh # from-param: ${app1}
spec:
  containers:
    - name: getting-started
      image: skaffold-example`,
				"values.env": `app1=from-file`,
			}, expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    a: from-command-line
spec:
  containers:
    - name: getting-started
      image: skaffold-example
`},
		{
			description: "test set param with helm native template(replicaCount)",
			args:        []string{"--offline=true", "--set", "replicaCount=5"},
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  helm:
    releases:
      - name: skaffold-helm
        chartPath: charts
        namespace: helm-namespace
`,
			input: map[string]string{
				"charts/templates/deployments.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
`,
				"charts/Chart.yaml": `
apiVersion: v1
description: Skaffold example with Helm
name: skaffold-helm
version: 0.1.0
`,
				"charts/values.yaml": `
replicaCount: 2
image: skaffold-helm:latest
`,
			}, expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  namespace: helm-namespace
  labels:
    app: skaffold-helm
spec:
  selector:
    matchLabels:
      app: skaffold-helm
  replicas: 5
  template:
    metadata:
      labels:
        app: skaffold-helm
    spec:
      containers:
      - name: skaffold-helm
        image: skaffold-helm:latest
`},
		{
			description:        "test set param with helm #from-param overrides native template(replicaCount) is comment templated field provided",
			args:               []string{"--offline=true", "--set", "replicaCount=5", "--set", "count=6"},
			WithBuildArtifacts: true,
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  helm:
    releases:
      - name: skaffold-helm
        chartPath: charts
        namespace: helm-namespace
`,
			input: map[string]string{
				"charts/templates/deployments.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  replicas: {{ .Values.replicaCount }} # from-param: ${count}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
`,
				"charts/Chart.yaml": `
apiVersion: v1
description: Skaffold example with Helm
name: skaffold-helm
version: 0.1.0
`,
				"charts/values.yaml": `
replicaCount: 2
image: skaffold-helm:latest
`,
			}, expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  namespace: helm-namespace
  labels:
    app: skaffold-helm
spec:
  selector:
    matchLabels:
      app: skaffold-helm
  replicas: 6
  template:
    metadata:
      labels:
        app: skaffold-helm
    spec:
      containers:
      - name: skaffold-helm
        image: skaffold-helm:latest
`},
		{
			description:        "test set param with helm #from-param has no effect on native template(replicaCount) when comment templated field value not provided",
			args:               []string{"--offline=true", "--set", "replicaCount=5"},
			WithBuildArtifacts: true,
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  helm:
    releases:
      - name: skaffold-helm
        chartPath: charts
        namespace: helm-namespace
`,
			input: map[string]string{
				"charts/templates/deployments.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  replicas: {{ .Values.replicaCount }} # from-param: ${count}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
`,
				"charts/Chart.yaml": `
apiVersion: v1
description: Skaffold example with Helm
name: skaffold-helm
version: 0.1.0
`,
				"charts/values.yaml": `
replicaCount: 2
image: skaffold-helm:latest
`,
			}, expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  namespace: helm-namespace
  labels:
    app: skaffold-helm
spec:
  selector:
    matchLabels:
      app: skaffold-helm
  replicas: 5
  template:
    metadata:
      labels:
        app: skaffold-helm
    spec:
      containers:
      - name: skaffold-helm
        image: skaffold-helm:latest
`},
		{
			description:        "test set param with helm #from-param, values are provided through file",
			args:               []string{"--offline=true", "--set-value-file", "values.env"},
			WithBuildArtifacts: true,
			config: `
apiVersion: skaffold/v4beta2
kind: Config
manifests:
  helm:
    releases:
      - name: skaffold-helm
        chartPath: charts
        namespace: helm-namespace
`,
			input: map[string]string{
				"charts/templates/deployments.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  replicas: {{ .Values.replicaCount }} # from-param: ${count}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }}
`,
				"charts/Chart.yaml": `
apiVersion: v1
description: Skaffold example with Helm
name: skaffold-helm
version: 0.1.0
`,
				"charts/values.yaml": `
replicaCount: 2
image: skaffold-helm:latest
`,
				"values.env": `count=3`,
			}, expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  namespace: helm-namespace
  labels:
    app: skaffold-helm
spec:
  selector:
    matchLabels:
      app: skaffold-helm
  replicas: 3
  template:
    metadata:
      labels:
        app: skaffold-helm
    spec:
      containers:
      - name: skaffold-helm
        image: skaffold-helm:latest
`},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			tmpDir := t.NewTempDir()
			tmpDir.Write("skaffold.yaml", test.config)

			for filePath, content := range test.input {
				tmpDir.Write(filePath, content)
			}

			if test.WithBuildArtifacts {
				dir, _ := os.Getwd()
				test.args = append(test.args, "--build-artifacts="+filepath.Join(dir, "testdata/render/build-output.json"))
			}

			tmpDir.Chdir()
			output := skaffold.Render(test.args...).RunOrFailOutput(t.T)

			t.CheckDeepEqual(test.expectedOut, string(output), testutil.YamlObj(t.T))
		})
	}
}

func TestRenderWithTagFlag(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description    string
		projectDir     string
		expectedOutput string
		namespaceFlag  string
	}{
		{
			description: "tag flag in a single module project",
			projectDir:  "testdata/getting-started",
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold-example:customtag
    name: getting-started
`,
		},
		{
			description: "tag flag in a multi module project",
			projectDir:  "testdata/multi-config-pods",
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: module1
spec:
  containers:
  - image: gcr.io/k8s-skaffold/multi-config-module1:customtag
    name: module1
---
apiVersion: v1
kind: Pod
metadata:
  name: module2
spec:
  containers:
  - image: gcr.io/k8s-skaffold/multi-config-module2:customtag
    name: module2
`,
		},
		{
			description:   "tag flag, with namespace flag, in a single module project",
			projectDir:    "testdata/getting-started",
			namespaceFlag: "mynamespace",
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  namespace: mynamespace
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold-example:customtag
    name: getting-started
`,
		},
		{
			description:   "tag flag, with namespace flag, in a multi module project",
			projectDir:    "testdata/multi-config-pods",
			namespaceFlag: "mynamespace",
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: module1
  namespace: mynamespace
spec:
  containers:
  - image: gcr.io/k8s-skaffold/multi-config-module1:customtag
    name: module1
---
apiVersion: v1
kind: Pod
metadata:
  name: module2
  namespace: mynamespace
spec:
  containers:
  - image: gcr.io/k8s-skaffold/multi-config-module2:customtag
    name: module2
`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := []string{"--tag", "customtag", "--default-repo", "gcr.io/k8s-skaffold"}

			if test.namespaceFlag != "" {
				args = append(args, "--namespace", test.namespaceFlag)
			}

			output := skaffold.Render(args...).InDir(test.projectDir).RunOrFailOutput(t.T)
			t.CheckDeepEqual(string(output), test.expectedOutput, testutil.YamlObj(t.T))
		})
	}
}

func TestKptRender(t *testing.T) {
	tests := []struct {
		description string
		args        []string
		config      string
		input       map[string]string // file path => content
		expectedOut string
	}{
		{
			description: "simple kpt render",
			config: `apiVersion: skaffold/v4beta2
kind: Config
metadata:
  name: getting-started-kustomize
manifests:
  kpt:
    - apply-simple
`, input: map[string]string{"apply-simple/Kptfile": `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: apply-setters-simple
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt-functions-catalog
    directory: /examples/apply-setters-simple
    ref: apply-setters/v0.2.0
  updateStrategy: resource-merge
upstreamLock:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt-functions-catalog
    directory: /examples/apply-setters-simple
    ref: apply-setters/v0.2.0
    commit: 9b6ce80e355a53727d21b2b336f8da55e760e20ca
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2
      configMap:
        nginx-replicas: 3
        tag: 1.16.2
`, "apply-simple/resources.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
spec:
  replicas: 4 # kpt-set: ${nginx-replicas}
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.1" # kpt-set: nginx:${tag}
          ports:
            - protocol: TCP
              containerPort: 80`},
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.2"
          ports:
            - protocol: TCP
              containerPort: 80
`},
		{
			description: "kpt render with data config file",
			config: `apiVersion: skaffold/v4beta2
kind: Config
metadata:
  name: getting-started-kustomize
manifests:
  kpt:
    - set-annotations
`,
			input: map[string]string{"set-annotations/Kptfile": `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1.4
      configPath: fn-config.yaml`,
				"set-annotations/fn-config.yaml": `apiVersion: fn.kpt.dev/v1alpha1
kind: SetAnnotations
metadata: # kpt-merge: /my-func-config
  name: my-func-config
  annotations:
    config.kubernetes.io/local-config: "true"
    internal.kpt.dev/upstream-identifier: 'fn.kpt.dev|SetAnnotations|default|my-func-config'
annotations:
  color: orange
  fruit: apple
`,
				"set-annotations/resources.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
spec:
  replicas: 3 # kpt-set: ${nginx-replicas}
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.1" # kpt-set: nginx:${tag}
          ports:
            - protocol: TCP
              containerPort: 80`},
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
  annotations: 
    color: orange
    fruit: apple
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      annotations: 
        color: orange
        fruit: apple
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.1"
          ports:
            - protocol: TCP
              containerPort: 80
`},
		{
			description: "simple kpt render with namespace flag",
			args:        []string{"--namespace", "mynamespace"},
			config: `apiVersion: skaffold/v4beta2
kind: Config
metadata:
  name: getting-started-kustomize
manifests:
  kpt:
  - apply-simple
`,
			input: map[string]string{"apply-simple/Kptfile": `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: apply-setters-simple
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt-functions-catalog
    directory: /examples/apply-setters-simple
    ref: apply-setters/v0.2.0
  updateStrategy: resource-merge
upstreamLock:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt-functions-catalog
    directory: /examples/apply-setters-simple
    ref: apply-setters/v0.2.0
    commit: 9b6ce80e355a53727d21b2b336f8da55e760e20ca
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2
      configMap:
        nginx-replicas: 3
        tag: 1.16.2
`,
				"apply-simple/resources.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
spec:
  replicas: 4 # kpt-set: ${nginx-replicas}
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.1" # kpt-set: nginx:${tag}
          ports:
            - protocol: TCP
              containerPort: 80
`},
			expectedOut: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
  namespace: mynamespace
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.16.2"
          ports:
            - protocol: TCP
              containerPort: 80
`},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			tmpDir := t.NewTempDir()
			tmpDir.Write("skaffold.yaml", test.config)

			for filePath, content := range test.input {
				tmpDir.Write(filePath, content)
			}

			tmpDir.Chdir()
			output := skaffold.Render(test.args...).RunOrFailOutput(t.T)
			var out map[string]any
			var ex map[string]any
			err := yaml.Unmarshal(output, &out)
			if err != nil {
				return
			}
			err = yaml.Unmarshal([]byte(test.expectedOut), &ex)
			if err != nil {
				return
			}

			t.CheckDeepEqual(ex, out)
		})
	}
}

func TestHelmRenderWithImagesFlag(t *testing.T) {
	tests := []struct {
		description  string
		dir          string
		args         []string
		builds       []graph.Artifact
		helmReleases []latest.HelmRelease
		expectedOut  string
	}{
		{
			description: "verify --images flag work with helm render",
			dir:         "testdata/helm-render",
			args:        []string{"--profile=helm-render", "--images=us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold-helm:latest@sha256:3e8981b13fadcbb5f4d42d00fdf52a9de128feea5280f0a1f7fb542cf31f1a06"},
			expectedOut: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm-skaffold-helm
spec:
  ports:
  - name: nginx
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: skaffold-helm
    release: skaffold-helm
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
    skaffold.dev/run-id: phony-run-id
  name: skaffold-helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: skaffold-helm
      release: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        release: skaffold-helm
        skaffold.dev/run-id: phony-run-id
    spec:
      containers:
      - image: us-central1-docker.pkg.dev/k8s-skaffold/testing/skaffold-helm:latest@sha256:3e8981b13fadcbb5f4d42d00fdf52a9de128feea5280f0a1f7fb542cf31f1a06
        imagePullPolicy: always
        name: skaffold-helm
        ports:
        - containerPort: 80
        resources: {}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations: null
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    heritage: Helm
    release: skaffold-helm
  name: skaffold-helm-skaffold-helm
spec:
  rules:
  - http:
      paths:
      - backend:
          service:
            name: skaffold-helm-skaffold-helm
            port:
              number: 80
        path: /
        pathType: ImplementationSpecific
`,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			out := skaffold.Render(append([]string{"--default-repo=", "--label=skaffold.dev/run-id=phony-run-id"}, test.args...)...).InDir(test.dir).RunOrFailOutput(t)

			testutil.CheckDeepEqual(t, test.expectedOut, string(out), testutil.YamlObj(t))
		})
	}
}
