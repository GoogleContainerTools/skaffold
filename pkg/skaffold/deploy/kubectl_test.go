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

package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
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
	tests := []struct {
		description          string
		cfg                  *latest.KubectlDeploy
		builds               []build.Artifact
		command              util.Command
		shouldErr            bool
		forceDeploy          bool
		expectedDependencies []string
	}{
		{
			description: "no manifest",
			cfg:         &latest.KubectlDeploy{},
			command:     testutil.FakeRunOut(t, "kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "missing manifest file",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"missing.yaml"},
			},
			command: testutil.FakeRunOut(t, "kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "ignore non-manifest",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"*.ignored"},
			},
			command: testutil.FakeRunOut(t, "kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "deploy success (forced)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			forceDeploy:          true,
			expectedDependencies: []string{"deployment.yaml"},
		},
		{
			description: "deploy success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			expectedDependencies: []string{"deployment.yaml"},
		},
		{
			description: "http manifest",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml", "http://remote.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml -f http://remote.yaml", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			expectedDependencies: []string{"deployment.yaml"},
		},
		{
			description: "deploy command error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			shouldErr:            true,
			expectedDependencies: []string{"deployment.yaml"},
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
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace apply -v=0 --overwrite=true -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			shouldErr:            true,
			expectedDependencies: []string{"deployment.yaml"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			t.NewTempDir().
				Write("deployment.yaml", deploymentWebYAML).
				Touch("empty.ignored").
				Chdir()

			k := NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
					Force:     test.forceDeploy,
				},
			})

			dependencies, err := k.Dependencies()
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedDependencies, dependencies)

			err = k.Deploy(context.Background(), ioutil.Discard, test.builds, nil).GetError()
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	tests := []struct {
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
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
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
				WithRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace delete -v=0 --grace-period=1 --ignore-not-found=true -f -"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			t.NewTempDir().
				Write("deployment.yaml", deploymentWebYAML).
				Chdir()

			k := NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			})
			err := k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlDeployerRemoteCleanup(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest.KubectlDeploy
		command     util.Command
	}{
		{
			description: "cleanup success",
			cfg: &latest.KubectlDeploy{
				RemoteManifests: []string{"pod/leeroy-web"},
			},
			command: testutil.NewFakeCmd(t).
				WithRun("kubectl --context kubecontext --namespace testNamespace get pod/leeroy-web -o yaml").
				WithRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -").
				WithRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", deploymentWebYAML),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				RemoteManifests: []string{"anotherNamespace:pod/leeroy-web"},
			},
			command: testutil.NewFakeCmd(t).
				WithRun("kubectl --context kubecontext --namespace anotherNamespace get pod/leeroy-web -o yaml").
				WithRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -").
				WithRunInput("kubectl --context kubecontext --namespace anotherNamespace apply -f -", deploymentWebYAML),
		},
	}
	for _, test := range tests {
		testutil.Run(t, "cleanup remote", func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			t.NewTempDir().
				Write("deployment.yaml", deploymentWebYAML).
				Chdir()

			k := NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			})
			err := k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(false, err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("deployment-web.yaml", deploymentWebYAML).
			Write("deployment-app.yaml", deploymentAppYAML)

		t.Override(&util.DefaultExecCommand, t.
			FakeRunOut("kubectl version --client -ojson", kubectlVersion).
			WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			WithRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v1
    name: leeroy-app
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`).
			WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			WithRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v2
    name: leeroy-app`).
			WithRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML),
		)

		cfg := &latest.KubectlDeploy{
			Manifests: []string{tmpDir.Path("deployment-app.yaml"), "deployment-web.yaml"},
		}
		deployer := NewKubectlDeployer(&runcontext.RunContext{
			WorkingDir: tmpDir.Root(),
			Cfg: latest.Pipeline{
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: cfg,
					},
				},
			},
			KubeContext: testKubeContext,
			Opts: config.SkaffoldOptions{
				Namespace: testNamespace,
			},
		})
		labellers := []Labeller{deployer}

		// Deploy one manifest
		err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
		}, labellers).GetError()
		t.CheckNoError(err)

		// Deploy one manifest since only one image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, labellers).GetError()
		t.CheckNoError(err)

		// Deploy zero manifest since no image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, labellers).GetError()
		t.CheckNoError(err)
	})
}
