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
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

const testKubeContext = "kubecontext"

const deploymentWebYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const deploymentAppYaml = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - name: leeroy-app
    image: leeroy-app`

func TestKubectlDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *latest.KubectlDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "parameter mismatch",
			shouldErr:   true,
			cfg: &latest.KubectlDeploy{
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
			cfg: &latest.KubectlDeploy{
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
			cfg: &latest.KubectlDeploy{
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
			cfg: &latest.KubectlDeploy{
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
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latest.KubectlFlags{
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

	tmpDir.Write("deployment.yaml", deploymentWebYAML)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace)
			_, err := k.Deploy(context.Background(), ioutil.Discard, test.builds)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *latest.KubectlDeploy
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", nil),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command:   testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "additional flags",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latest.KubectlFlags{
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

	tmpDir.Write("deployment.yaml", deploymentWebYAML)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace)
			err := k.Cleanup(context.Background(), ioutil.Discard)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd("kubectl --context kubecontext --namespace testNamespace apply -f -", nil)

	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("deployment-web.yaml", deploymentWebYAML)
	tmpDir.Write("deployment-app.yaml", deploymentAppYaml)

	cfg := &latest.KubectlDeploy{
		Manifests: []string{"deployment-web.yaml", "deployment-app.yaml"},
	}
	deployer := NewKubectlDeployer(tmpDir.Root(), cfg, testKubeContext, testNamespace)

	// Deploy one manifest
	deployed, err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, 2, len(deployed))

	// Deploy one manifest since only one image is updated
	deployed, err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, 1, len(deployed))

	// Deploy zero manifest since no image is updated
	deployed, err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, 0, len(deployed))
}
