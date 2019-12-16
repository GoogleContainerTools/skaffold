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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	testKubeContext = "kubecontext"
	testKubeConfig  = "kubeconfig"
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
		description string
		cfg         *latest.KubectlDeploy
		builds      []build.Artifact
		commands    util.Command
		shouldErr   bool
		forceDeploy bool
	}{
		{
			description: "no manifest",
			cfg:         &latest.KubectlDeploy{},
			commands:    testutil.CmdRunOut("kubectl version --client -ojson", kubectlVersion),
		},
		{
			description: "deploy success (forced)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			forceDeploy: true,
		},
		{
			description: "deploy success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
		},
		{
			description: "http manifest",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml", "http://remote.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml -f http://remote.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
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
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")),
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
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -v=0 --overwrite=true -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
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

			err := k.Deploy(context.Background(), ioutil.Discard, test.builds, nil).GetError()

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest.KubectlDeploy
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
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
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete -v=0 --grace-period=1 --ignore-not-found=true -f -"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
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
		commands    util.Command
	}{
		{
			description: "cleanup success",
			cfg: &latest.KubectlDeploy{
				RemoteManifests: []string{"pod/leeroy-web"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace get pod/leeroy-web -o yaml").
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -").
				AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", deploymentWebYAML),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				RemoteManifests: []string{"anotherNamespace:pod/leeroy-web"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace anotherNamespace get pod/leeroy-web -o yaml").
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -").
				AndRunInput("kubectl --context kubecontext --namespace anotherNamespace apply -f -", deploymentWebYAML),
		},
	}
	for _, test := range tests {
		testutil.Run(t, "cleanup remote", func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
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

			t.CheckNoError(err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("deployment-web.yaml", deploymentWebYAML).
			Write("deployment-app.yaml", deploymentAppYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", kubectlVersion).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", `apiVersion: v1
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
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v2
    name: leeroy-app`).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML),
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

		// Deploy one manifest
		err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
		}, nil).GetError()
		t.CheckNoError(err)

		// Deploy one manifest since only one image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, nil).GetError()
		t.CheckNoError(err)

		// Deploy zero manifest since no image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, nil).GetError()
		t.CheckNoError(err)
	})
}

func TestDependencies(t *testing.T) {
	tests := []struct {
		description string
		manifests   []string
		expected    []string
	}{
		{
			description: "no manifest",
			manifests:   []string(nil),
			expected:    []string(nil),
		},
		{
			description: "missing manifest file",
			manifests:   []string{"missing.yaml"},
			expected:    []string(nil),
		},
		{
			description: "ignore non-manifest",
			manifests:   []string{"*.ignored"},
			expected:    []string(nil),
		},
		{
			description: "single manifest",
			manifests:   []string{"deployment.yaml"},
			expected:    []string{"deployment.yaml"},
		},
		{
			description: "keep manifests order",
			manifests:   []string{"01_name.yaml", "00_service.yaml"},
			expected:    []string{"01_name.yaml", "00_service.yaml"},
		},
		{
			description: "sort children",
			manifests:   []string{"01/*.yaml", "00/*.yaml"},
			expected:    []string{filepath.Join("01", "a.yaml"), filepath.Join("01", "b.yaml"), filepath.Join("00", "a.yaml"), filepath.Join("00", "b.yaml")},
		},
		{
			description: "http manifest",
			manifests:   []string{"deployment.yaml", "http://remote.yaml"},
			expected:    []string{"deployment.yaml"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Touch("deployment.yaml", "01_name.yaml", "00_service.yaml", "empty.ignored").
				Touch("01/a.yaml", "01/b.yaml").
				Touch("00/b.yaml", "00/a.yaml").
				Chdir()

			k := NewKubectlDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: test.manifests,
							},
						},
					},
				},
			})
			dependencies, err := k.Dependencies()

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, dependencies)
		})
	}
}

func TestKubectlRender(t *testing.T) {
	tests := []struct {
		description string
		builds      []build.Artifact
		input       string
	}{
		{
			description: "normal render",
			builds: []build.Artifact{
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
		},
		{
			description: "two artifacts",
			builds: []build.Artifact{
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
		spec:
		  containers:
		  - image: gcr.io/project/image1
		    name: image1
		  - image: gcr.io/project/image2
		    name: image2
		`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f deployment.yaml", test.input))

			deployer := NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: []string{"deployment.yaml"},
							},
						},
					},
				},
				KubeContext: testKubeContext,
			})
			var b bytes.Buffer
			err := deployer.Render(context.Background(), &b, test.builds, "")
			t.CheckNoError(err)
		})
	}
}
