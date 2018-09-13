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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
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
		cfg         *v1alpha3.KubectlDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "parameter mismatch",
			shouldErr:   true,
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
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
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
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
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace apply -f -", nil),
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
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")),
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
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: v1alpha3.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"--overwrite=true"},
					Delete: []string{"ignored"},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace -v=0 apply -f -", fmt.Errorf("")),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
	}

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("deployment.yaml", deploymentYAML)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace)
			_, err := k.Deploy(context.Background(), &bytes.Buffer{}, test.builds)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *v1alpha3.KubectlDeploy
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", nil),
		},
		{
			description: "cleanup error",
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command:   testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "additional flags",
			cfg: &v1alpha3.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: v1alpha3.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"ignored"},
					Delete: []string{"--grace-period=1"},
				},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace -v=0 delete --grace-period=1 --ignore-not-found=true -f -", nil),
		},
	}

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("deployment.yaml", deploymentYAML)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace)
			err := k.Cleanup(context.Background(), &bytes.Buffer{})

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
