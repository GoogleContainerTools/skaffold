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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectlRenderOutput(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	test := struct {
		description string
		builds      []graph.Artifact
		renderPath  string
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
		renderPath: "./test-output",
		input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
		expectedOut: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
    name: skaffold
`}

	testutil.Run(t, test.description, func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Write("deployment.yaml", test.input).Chdir()

		deployer, err := kubectl.NewDeployer(&v2.RunContext{
			WorkingDir: ".",
			Pipelines: v2.NewPipelines([]latestV2.Pipeline{{
				Render: latestV2.RenderConfig{
					Generate: latestV2.Generate{
						RawK8s: []string{"deployment.yaml"}},
				},
			}}),
		}, &label.DefaultLabeller{}, &latestV2.KubectlDeploy{
			Manifests: []string{"deployment.yaml"},
		}, filepath.Join(tmpDir.Root(), test.renderPath))
		t.RequireNoError(err)
		var b bytes.Buffer
		err = deployer.Render(context.Background(), &b, test.builds, false, test.renderPath)

		t.CheckNoError(err)
		dat, err := ioutil.ReadFile(test.renderPath)
		t.CheckNoError(err)

		t.CheckDeepEqual(test.expectedOut, string(dat))
	})
}

func TestKubectlRender(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

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
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
    name: skaffold
`,
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
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
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
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod-123
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  name: my-pod-456
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Write("deployment.yaml", test.input).
				Chdir()

			deployer, err := kubectl.NewDeployer(&v2.RunContext{
				WorkingDir: ".",
				Pipelines: v2.NewPipelines([]latestV2.Pipeline{{
					Render: latestV2.RenderConfig{
						Generate: latestV2.Generate{
							RawK8s: []string{"deployment.yaml"}},
					},
				}}),
				Opts: config.SkaffoldOptions{},
			}, &label.DefaultLabeller{}, &latestV2.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			}, "")
			t.RequireNoError(err)
			var b bytes.Buffer
			err = deployer.Render(context.Background(), &b, test.builds, false, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, b.String())
		})
	}
}

func TestHelmRender(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		builds       []graph.Artifact
		helmReleases []latestV2.HelmRelease
		expectedOut  string
	}{
		{
			description: "Bare bones render",
			builds: []graph.Artifact{
				{
					ImageName: "gke-loadbalancer",
					Tag:       "gke-loadbalancer:test",
				},
			},
			helmReleases: []latestV2.HelmRelease{{
				Name:      "gke-loadbalancer",
				ChartPath: "testdata/gke_loadbalancer/loadbalancer-helm",
			}},
			expectedOut: `---
# Source: loadbalancer-helm/templates/k8s.yaml
apiVersion: v1
kind: Service
metadata:
  name: gke-loadbalancer
  labels:
    app: gke-loadbalancer
spec:
  type: LoadBalancer
  ports:
    - port: 80
      targetPort: 3000
      protocol: TCP
      name: http
  selector:
    app: "gke-loadbalancer"
---
# Source: loadbalancer-helm/templates/k8s.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gke-loadbalancer
  labels:
    app: gke-loadbalancer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gke-loadbalancer
  template:
    metadata:
      labels:
        app: gke-loadbalancer
    spec:
      containers:
        - name: gke-container
          image: gke-loadbalancer:test
          ports:
            - containerPort: 3000

`,
		},
		{
			description: "A more complex template",
			builds: []graph.Artifact{
				{
					ImageName: "gcr.io/k8s-skaffold/skaffold-helm",
					Tag:       "gcr.io/k8s-skaffold/skaffold-helm:sha256-nonsenslettersandnumbers",
				},
			},
			helmReleases: []latestV2.HelmRelease{{
				Name:      "skaffold-helm",
				ChartPath: "testdata/helm/skaffold-helm",
				SetValues: map[string]string{
					"pullPolicy": "Always",
				},
			}},
			expectedOut: `---
# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
      name: nginx
  selector:
    app: skaffold-helm
    release: skaffold-helm
---
# Source: skaffold-helm/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
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
    spec:
      containers:
        - name: skaffold-helm
          image: gcr.io/k8s-skaffold/skaffold-helm:sha256-nonsenslettersandnumbers
          imagePullPolicy: Always
          ports:
            - containerPort: 80
          resources:
            {}
---
# Source: skaffold-helm/templates/ingress.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
  annotations:
spec:
  rules:
    - http:
        paths:
          - path: /
            backend:
              serviceName: skaffold-helm-skaffold-helm
              servicePort: 80

`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deployer, err := helm.NewDeployer(context.Background(), &v2.RunContext{
				Pipelines: v2.NewPipelines([]latestV2.Pipeline{{
					Deploy: latestV2.DeployConfig{
						DeployType: latestV2.DeployType{
							LegacyHelmDeploy: &latestV2.LegacyHelmDeploy{
								Releases: test.helmReleases,
							},
						},
					},
				}}),
			}, &label.DefaultLabeller{}, &latestV2.LegacyHelmDeploy{
				Releases: test.helmReleases,
			}, nil)
			t.RequireNoError(err)
			var b bytes.Buffer
			err = deployer.Render(context.Background(), &b, test.builds, true, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, b.String())
		})
	}
}

func TestRenderWithBuilds(t *testing.T) {
	// TODO: This test shall pass once render v2 is completed.
	t.SkipNow()

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
				args = append(args, "--offline")
				skaffold.Render(args...).WithEnv(env).RunOrFail(t.T)
			} else {
				skaffold.Render(args...).RunOrFail(t.T)
			}

			fileContent, err := ioutil.ReadFile("rendered.yaml")
			t.RequireNoError(err)

			// Tests are written in a way that actual output is valid YAML
			parsed := make(map[string]interface{})
			err = yaml.UnmarshalStrict(fileContent, parsed)
			t.CheckNoError(err)

			fileContentReplaced := regexp.MustCompile("(?m)(skaffold.dev/run-id|skaffold.dev/docker-api-version): .+$").ReplaceAll(fileContent, []byte("$1: SOMEDYNAMICVALUE"))

			t.RequireNoError(err)
			t.CheckDeepEqual(test.expectedOut, string(fileContentReplaced))
		})
	}
}
