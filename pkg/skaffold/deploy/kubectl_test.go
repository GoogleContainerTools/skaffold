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

const (
	testKubeContext = "kubecontext"
	kubectlVersion  = `{"clientVersion":{"major":"1","minor":"12"}}`
)

const deploymentWebYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const deploymentAppYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - name: leeroy-app
    image: leeroy-app`

func TestKubectlDeploy(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("deployment.yaml", deploymentWebYAML)
	tmpDir.Write("empty.ignored", "")

	var tests = []struct {
		description string
		cfg         *latest.KubectlDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "no manifest",
			cfg:         &latest.KubectlDeploy{},
			command:     testutil.NewFakeCmd(t).WithRunOut("kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "missing manifest file",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"missing.yaml"},
			},
			command: testutil.NewFakeCmd(t).WithRunOut("kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "ignore non-manifest",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"*.ignored"},
			},
			command: testutil.NewFakeCmd(t).WithRunOut("kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "deploy success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace apply --force -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
		},
		{
			description: "deploy command error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace apply --force -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			shouldErr: true,
		},
		{
			description: "additional flags",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latest.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"--overwrite=true"},
					Delete: []string{"ignored"},
				},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace -v=0 create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace -v=0 apply --overwrite=true --force -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = test.command

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace, "")
			err := k.Deploy(context.Background(), ioutil.Discard, test.builds, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tmpDir.Write("deployment.yaml", deploymentWebYAML)

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
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
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
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace -v=0 create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace -v=0 delete --grace-period=1 --ignore-not-found=true -f -"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = test.command

			k := NewKubectlDeployer(tmpDir.Root(), test.cfg, testKubeContext, testNamespace, "")
			err := k.Cleanup(context.Background(), ioutil.Discard)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()
	tmpDir.Write("deployment-web.yaml", deploymentWebYAML)
	tmpDir.Write("deployment-app.yaml", deploymentAppYAML)

	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).
		WithRunOut("kubectl version --client -ojson", kubectlVersion).
		WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
		WithRunInput("kubectl --context kubecontext --namespace testNamespace apply --force -f -", `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v1
    name: leeroy-app
---
apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`).
		WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
		WithRunInput("kubectl --context kubecontext --namespace testNamespace apply --force -f -", `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v2
    name: leeroy-app`).
		WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML)

	cfg := &latest.KubectlDeploy{
		Manifests: []string{"*.yaml"},
	}
	deployer := NewKubectlDeployer(tmpDir.Root(), cfg, testKubeContext, testNamespace, "")

	// Deploy one manifest
	err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
	}, nil)
	testutil.CheckError(t, false, err)

	// Deploy one manifest since only one image is updated
	err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
	}, nil)
	testutil.CheckError(t, false, err)

	// Deploy zero manifest since no image is updated
	err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
		{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
	}, nil)
	testutil.CheckError(t, false, err)
}
