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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	testKubeContext   = "kubecontext"
	testKubeConfig    = "kubeconfig"
	kubectlVersion112 = `{"clientVersion":{"major":"1","minor":"12"}}`
	kubectlVersion118 = `{"clientVersion":{"major":"1","minor":"18"}}`
)

const deploymentWebYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const deploymentWebYAMLv1 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`

const deploymentAppYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - name: leeroy-app
    image: leeroy-app`

const deploymentAppYAMLv1 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v1
    name: leeroy-app`

const deploymentAppYAMLv2 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v2
    name: leeroy-app`

func TestKubectlDeploy(t *testing.T) {
	tests := []struct {
		description      string
		cfg              *latest.KubectlDeploy
		builds           []build.Artifact
		commands         util.Command
		shouldErr        bool
		forceDeploy      bool
		waitForDeletions bool
	}{
		{
			description:      "no manifest",
			cfg:              &latest.KubectlDeploy{},
			commands:         testutil.CmdRunOut("kubectl version --client -ojson", kubectlVersion112),
			waitForDeletions: true,
		},
		{
			description: "deploy success (disable validation)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latest.KubectlFlags{
					DisableValidation: true,
				},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml --validate=false", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --validate=false"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (forced)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:      true,
			waitForDeletions: true,
		},
		{
			description: "deploy success",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (kubectl v1.18)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run=client -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "http manifest",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml", "http://remote.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml -f http://remote.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
		},
		{
			description: "deploy command error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
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
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -v=0 -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -v=0 --overwrite=true -f -", fmt.Errorf("")),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
			shouldErr:        true,
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
					WaitForDeletions: config.WaitForDeletions{
						Enabled: test.waitForDeletions,
						Delay:   0 * time.Second,
						Max:     10 * time.Second,
					},
				},
			}, nil)

			_, err := k.Deploy(context.Background(), ioutil.Discard, test.builds)

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
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup success (kubectl v1.18)",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run=client -oyaml -f deployment.yaml", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
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
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", deploymentWebYAML).
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
			}, nil)
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
			}, nil)
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
			CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentAppYAMLv1+"\n---\n"+deploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", deploymentAppYAMLv1+"\n---\n"+deploymentWebYAMLv1).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentAppYAMLv2+"\n---\n"+deploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", deploymentAppYAMLv2).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), deploymentAppYAML+"\n"+deploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentAppYAMLv2+"\n---\n"+deploymentWebYAMLv1, ""),
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
				WaitForDeletions: config.WaitForDeletions{
					Enabled: true,
					Delay:   0 * time.Millisecond,
					Max:     10 * time.Second,
				},
			},
		}, nil)

		// Deploy one manifest
		_, err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
		})
		t.CheckNoError(err)

		// Deploy one manifest since only one image is updated
		_, err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		})
		t.CheckNoError(err)

		// Deploy zero manifest since no image is updated
		_, err = deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		})
		t.CheckNoError(err)
	})
}

func TestKubectlWaitForDeletions(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", deploymentWebYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-web.yaml"), deploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", deploymentWebYAMLv1),
		)

		cfg := &latest.KubectlDeploy{
			Manifests: []string{tmpDir.Path("deployment-web.yaml")},
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
				WaitForDeletions: config.WaitForDeletions{
					Enabled: true,
					Delay:   0 * time.Millisecond,
					Max:     10 * time.Second,
				},
			},
		}, nil)

		var out bytes.Buffer
		_, err := deployer.Deploy(context.Background(), &out, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual(` - 2 resources are marked for deletion, waiting for completion: "leeroy-web", "leeroy-app"
 - "leeroy-web" is marked for deletion, waiting for completion
`, out.String())
	})
}

func TestKubectlWaitForDeletionsFails(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", deploymentWebYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
			AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+tmpDir.Path("deployment-web.yaml"), deploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}}
				]
			}`),
		)

		cfg := &latest.KubectlDeploy{
			Manifests: []string{tmpDir.Path("deployment-web.yaml")},
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
				WaitForDeletions: config.WaitForDeletions{
					Enabled: true,
					Delay:   10 * time.Second,
					Max:     100 * time.Millisecond,
				},
			},
		}, nil)

		_, err := deployer.Deploy(context.Background(), ioutil.Discard, []build.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		})

		t.CheckErrorContains(`2 resources failed to complete their deletion before a new deployment: "leeroy-web", "leeroy-app"`, err)
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
			}, nil)
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
		expected    string
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
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
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
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
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
			description: "no artifacts",
			builds:      nil,
			input: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: image1:tag1
    name: image1
  - image: image2:tag2
    name: image2
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("deployment.yaml", test.input)

			t.Override(&util.DefaultExecCommand, testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), test.input))
			defaultRepo := config.StringOrUndefined{}
			defaultRepo.Set("gcr.io/project")
			deployer := NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: []string{tmpDir.Path("deployment.yaml")},
							},
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					DefaultRepo: defaultRepo,
				},
			}, nil)
			var b bytes.Buffer
			err := deployer.Render(context.Background(), &b, test.builds, true, "")
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, b.String())
		})
	}
}

func TestGCSManifests(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest.KubectlDeploy
		commands    util.Command
		shouldErr   bool
		skipRender  bool
	}{
		{
			description: "manifest from GCS",
			cfg: &latest.KubectlDeploy{
				Manifests: []string{"gs://dev/deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut(fmt.Sprintf("gsutil cp -r %s %s", "gs://dev/deployment.yaml", manifestTmpDir), "log").
				AndRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+filepath.Join(manifestTmpDir, "deployment.yaml"), deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			skipRender: true,
		}}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			if err := os.MkdirAll(manifestTmpDir, os.ModePerm); err != nil {
				t.Fatal(err)
			}
			if err := ioutil.WriteFile(manifestTmpDir+"/deployment.yaml", []byte(deploymentWebYAML), os.ModePerm); err != nil {
				t.Fatal(err)
			}

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
					Namespace:  testNamespace,
					SkipRender: test.skipRender,
				},
			}, nil)

			_, err := k.Deploy(context.Background(), ioutil.Discard, nil)

			t.CheckError(test.shouldErr, err)
		})
	}
}
