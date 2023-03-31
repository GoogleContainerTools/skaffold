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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectlRenderOutput(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
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
				ImageName: "gcr.io/k8s-skaffold/skaffold",
				Tag:       "gcr.io/k8s-skaffold/skaffold:test",
			},
		},
		input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
		expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  namespace: %s
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
    name: skaffold`, ns.Name)}

	testutil.Run(t, test.description, func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Write("deployment.yaml", test.input).Chdir()

		rc := latest.RenderConfig{
			Generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"}},
		}
		mockCfg := render.MockConfig{WorkingDir: tmpDir.Root()}
		r, err := kubectl.New(mockCfg, rc, map[string]string{}, "default", ns.Name)
		t.RequireNoError(err)
		var b bytes.Buffer
		l, err := r.Render(context.Background(), &b, test.builds, false)

		t.CheckNoError(err)

		t.CheckDeepEqual(test.expectedOut, l.String(), testutil.YamlObj(t.T))
	})
}

func TestKubectlRender(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
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
					ImageName: "gcr.io/k8s-skaffold/skaffold",
					Tag:       "gcr.io/k8s-skaffold/skaffold:test",
				},
			},
			input: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
			expectedOut: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: %s
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
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
			tmpDir := t.NewTempDir()
			tmpDir.Write("deployment.yaml", test.input).
				Chdir()
			rc := latest.RenderConfig{
				Generate: latest.Generate{
					RawK8s: []string{"deployment.yaml"}},
			}
			mockCfg := render.MockConfig{WorkingDir: tmpDir.Root()}
			r, err := kubectl.New(mockCfg, rc, map[string]string{}, "default", ns.Name)
			t.RequireNoError(err)
			var b bytes.Buffer
			l, err := r.Render(context.Background(), &b, test.builds, false)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, l.String(), testutil.YamlObj(t.T))
		})
	}
}

func TestHelmRender(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

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
      - image: gcr.io/k8s-skaffold/skaffold-helm:sha256-nonsenselettersandnumbers
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
			description:      "Template with Release.namespace set from skaffold.yaml file",
			dir:              "testdata/helm-namespace",
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
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description         string
		config              string
		buildOutputFilePath string
		images              string
		offline             bool
		input               map[string]string // file path => content
		expectedOut         string
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
			// `metadata.namespace` is injected by `kubectl create` in non-offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: default
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
			// `metadata.namespace` is injected by `kubectl create` in non-offline mode
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: default
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
			description:              "project with rawYaml and transform should create hydration dir (uses kpt renderer)",
			shouldCreateHydrationDir: true,
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
