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
	"sort"
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

type fakeWarner struct {
	warnings []string
}

func (l *fakeWarner) Warnf(format string, args ...interface{}) {
	l.warnings = append(l.warnings, fmt.Sprintf(format, args...))
	sort.Strings(l.warnings)
}

func TestKubectlDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *v1alpha2.KubectlDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "parameter mismatch",
			shouldErr:   true,
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:v1",
				},
			},
		},
		{
			description: "missing manifest file",
			shouldErr:   true,
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "deploy success",
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext apply -f -", nil),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "deploy command error",
			shouldErr:   true,
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext apply -f -", fmt.Errorf("")),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "additional flags",
			shouldErr:   true,
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
				Flags: v1alpha2.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"--overwrite=true"},
					Delete: []string{"ignored"},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext -v=0 apply -f -", fmt.Errorf("")),
			builds: []build.Artifact{
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
			_, err := k.Deploy(context.Background(), &bytes.Buffer{}, test.builds)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *v1alpha2.KubectlDeploy
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext delete --ignore-not-found=true -f -", nil),
		},
		{
			description: "cleanup error",
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
			},
			command:   testutil.NewFakeCmd("kubectl --context kubecontext delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "additional flags",
			cfg: &v1alpha2.KubectlDeploy{
				Manifests: []string{"test/deployment.yaml"},
				Flags: v1alpha2.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"ignored"},
					Delete: []string{"--grace-period=1"},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext -v=0 delete --grace-period=1 --ignore-not-found=true -f -", nil),
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
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:latest
    name: latest
  - image: gcr.io/k8s-skaffold/example:v1
    name: fully-qualified
  - image: skaffold/other
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
  - image: skaffold/usedbyfqn:TAG
  - image: skaffold/usedwrongfqn:OTHER
`)}

	builds := []build.Artifact{{
		ImageName: "gcr.io/k8s-skaffold/example",
		Tag:       "gcr.io/k8s-skaffold/example:TAG",
	}, {
		ImageName: "skaffold/other",
		Tag:       "skaffold/other:OTHER_TAG",
	}, {
		ImageName: "skaffold/unused",
		Tag:       "skaffold/unused:TAG",
	}, {
		ImageName: "skaffold/usedbyfqn",
		Tag:       "skaffold/usedbyfqn:TAG",
	}, {
		ImageName: "skaffold/usedwrongfqn",
		Tag:       "skaffold/usedwrongfqn:TAG",
	}}

	expected := manifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:TAG
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:TAG
    name: latest
  - image: gcr.io/k8s-skaffold/example:v1
    name: fully-qualified
  - image: skaffold/other:OTHER_TAG
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
  - image: skaffold/usedbyfqn:TAG
  - image: skaffold/usedwrongfqn:OTHER
`)}

	defer func(w Warner) { warner = w }(warner)
	fakeWarner := &fakeWarner{}
	warner = fakeWarner

	resultManifest, err := manifests.replaceImages(builds)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
	testutil.CheckErrorAndDeepEqual(t, false, err, []string{
		"image [skaffold/unused] is not used by the deployment",
		"image [skaffold/usedwrongfqn] is not used by the deployment",
	}, fakeWarner.warnings)
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
