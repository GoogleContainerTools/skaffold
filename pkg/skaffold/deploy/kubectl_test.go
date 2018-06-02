/*
Copyright 2018 The Skaffold Authors

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

package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

const testKubeContext = "kubecontext"

const deploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-web
  labels:
    app: leeroy-web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-web
  template:
    metadata:
      labels:
        app: leeroy-web
    spec:
      containers:
      - name: leeroy-web
        image: leeroy-web
        ports:
        - containerPort: 8080`

func TestKubectlDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *v1alpha2.DeployConfig
		builds      []build.Build
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "parameter mismatch",
			shouldErr:   true,
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			builds: []build.Build{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:v1",
				},
			},
		},
		{
			description: "missing manifest file",
			shouldErr:   true,
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			builds: []build.Build{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "deploy success",
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext apply -f -", nil),
			builds: []build.Build{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "deploy command error",
			shouldErr:   true,
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext apply -f -", fmt.Errorf("")),
			builds: []build.Build{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
	}

	tmp, cleanup := testutil.TempDir(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(tmp, "test"), 0750)
	ioutil.WriteFile(filepath.Join(tmp, "test", "deployment.yaml"), []byte(deploymentYAML), 0644)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmp, test.cfg, testKubeContext)
			err := k.Deploy(context.Background(), &bytes.Buffer{}, test.builds)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *v1alpha2.DeployConfig
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext delete -f -", nil),
		},
		{
			description: "cleanup error",
			cfg: &v1alpha2.DeployConfig{
				DeployType: v1alpha2.DeployType{
					KubectlDeploy: &v1alpha2.KubectlDeploy{
						Manifests: []string{"test/deployment.yaml"},
					},
				},
			},
			command:   testutil.NewFakeCmd("kubectl --context kubecontext delete -f -", errors.New("BUG")),
			shouldErr: true,
		},
	}

	tmp, cleanup := testutil.TempDir(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(tmp, "test"), 0750)
	ioutil.WriteFile(filepath.Join(tmp, "test", "deployment.yaml"), []byte(deploymentYAML), 0644)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmp, test.cfg, testKubeContext)
			err := k.Cleanup(context.Background(), &bytes.Buffer{})

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestReplaceImages(t *testing.T) {
	manifests := manifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    key: value
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:latest
    name: latest
  - image: gcr.io/k8s-skaffold/example:v1
    name: fully-qualified`), []byte(`
apiVersion: v1
kind: Deployment
metadata:
  name: deployment
template:
  spec:
    containers:
    - image: skaffold/other
      name: other
    - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
      name: digest
`)}

	builds := []build.Build{{
		ImageName: "gcr.io/k8s-skaffold/example",
		Tag:       "gcr.io/k8s-skaffold/example:TAG",
	}, {
		ImageName: "skaffold/other",
		Tag:       "skaffold/other:OTHER_TAG",
	}}

	expected := manifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    key: value
    skaffold: "true"
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:TAG
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:TAG
    name: latest
  - image: gcr.io/k8s-skaffold/example:v1
    name: fully-qualified`), []byte(`
apiVersion: v1
kind: Deployment
metadata:
  labels:
    skaffold: "true"
  name: deployment
template:
  spec:
    containers:
    - image: skaffold/other:OTHER_TAG
      name: other
    - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
      name: digest
`)}

	resultManifest, err := manifests.replaceImages(builds)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestReplaceEmptyManifest(t *testing.T) {
	manifests := manifestList{[]byte(""), []byte("  ")}
	expected := manifestList{}

	resultManifest, err := manifests.replaceImages(nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestReplaceInvalidManifest(t *testing.T) {
	manifests := manifestList{[]byte("INVALID")}

	_, err := manifests.replaceImages(nil)

	testutil.CheckError(t, true, err)
}

func TestGenerateManifest(t *testing.T) {
	dockerfile, cleanup := testutil.TempFile(t, "Dockerfile", []byte("FROM scratch\nEXPOSE 80"))
	defer cleanup()

	builds := []build.Build{{
		ImageName: "gcr.io/k8s-skaffold/skaffold-example",
		Tag:       "gcr.io/k8s-skaffold/skaffold-example:TAG",
		Artifact: &v1alpha2.Artifact{
			Workspace: filepath.Dir(dockerfile),
			ArtifactType: v1alpha2.ArtifactType{
				DockerArtifact: &v1alpha2.DockerArtifact{
					DockerfilePath: filepath.Base(dockerfile),
				},
			},
		}},
	}

	deployer := &KubectlDeployer{
		DeployConfig: &v1alpha2.DeployConfig{
			DeployType: v1alpha2.DeployType{
				KubectlDeploy: &v1alpha2.KubectlDeploy{},
			},
		},
	}
	manifests, err := deployer.readOrGenerateManifests(builds)
	testutil.CheckError(t, false, err)

	manifests, err = manifests.replaceImages(builds)

	testutil.CheckErrorAndDeepEqual(t, false, err, `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    run: skaffold
    skaffold: "true"
  name: skaffold
spec:
  replicas: 1
  selector:
    matchLabels:
      run: skaffold
  strategy: {}
  template:
    metadata:
      labels:
        run: skaffold
    spec:
      containers:
      - image: gcr.io/k8s-skaffold/skaffold-example:TAG
        name: app
        ports:
        - containerPort: 80`, manifests.String())
}

func TestSplitTag(t *testing.T) {
	var tests = []struct {
		description            string
		image                  string
		expectedName           string
		expectedFullyQualified bool
	}{
		{
			description:            "port and tag",
			image:                  "host:1234/user/container:tag",
			expectedName:           "host:1234/user/container",
			expectedFullyQualified: true,
		},
		{
			description:            "port",
			image:                  "host:1234/user/container",
			expectedName:           "host:1234/user/container",
			expectedFullyQualified: false,
		},
		{
			description:            "tag",
			image:                  "host/user/container:tag",
			expectedName:           "host/user/container",
			expectedFullyQualified: true,
		},
		{
			description:            "latest",
			image:                  "host/user/container:latest",
			expectedName:           "host/user/container",
			expectedFullyQualified: false,
		},
		{
			description:            "digest",
			image:                  "gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
			expectedName:           "gcr.io/k8s-skaffold/example",
			expectedFullyQualified: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			parsed, err := parseReference(test.image)

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedName, parsed.baseName)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFullyQualified, parsed.fullyQualified)
		})
	}
}
