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

package kubectl

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectlDeploy(t *testing.T) {
	tests := []struct {
		description                 string
		kubectl                     latestV1.KubectlDeploy
		builds                      []graph.Artifact
		commands                    util.Command
		shouldErr                   bool
		forceDeploy                 bool
		waitForDeletions            bool
		skipSkaffoldNamespaceOption bool
		envs                        map[string]string
	}{
		{
			description:      "no manifest",
			kubectl:          latestV1.KubectlDeploy{},
			commands:         testutil.CmdRunOut("kubectl version --client -ojson", KubectlVersion112),
			waitForDeletions: true,
		},
		{
			description: "deploy success (disable validation)",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latestV1.KubectlFlags{
					DisableValidation: true,
				},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml --validate=false", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --validate=false"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (forced)",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:      true,
			waitForDeletions: true,
		},
		{
			description: "deploy success",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (kubectl v1.18)",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run=client -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (default namespace)",
			kubectl: latestV1.KubectlDeploy{
				Manifests:        []string{"deployment.yaml"},
				DefaultNamespace: &TestNamespace2,
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace2 create --dry-run=client -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f -"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions:            true,
			skipSkaffoldNamespaceOption: true,
		},
		{
			description: "deploy success (default namespace with env template)",
			kubectl: latestV1.KubectlDeploy{
				Manifests:        []string{"deployment.yaml"},
				DefaultNamespace: &TestNamespace2FromEnvTemplate,
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace2 create --dry-run=client -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f -"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions:            true,
			skipSkaffoldNamespaceOption: true,
			envs: map[string]string{
				"MYENV": "Namesp",
			},
		},
		{
			description: "http manifest",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml", "http://remote.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml -f http://remote.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy command error",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			shouldErr:        true,
			waitForDeletions: true,
		},
		{
			description: "additional flags",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latestV1.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"--overwrite=true"},
					Delete: []string{"ignored"},
				},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -v=0 -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -v=0 --overwrite=true -f -", fmt.Errorf("")),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			shouldErr:        true,
			waitForDeletions: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(test.envs)
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&client.Client, deployutil.MockK8sClient)
			t.NewTempDir().
				Write("deployment.yaml", DeploymentWebYAML).
				Touch("empty.ignored").
				Chdir()

			skaffoldNamespaceOption := ""
			if !test.skipSkaffoldNamespaceOption {
				skaffoldNamespaceOption = TestNamespace
			}

			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				force:      test.forceDeploy,
				waitForDeletions: config.WaitForDeletions{
					Enabled: test.waitForDeletions,
					Delay:   0 * time.Second,
					Max:     10 * time.Second,
				},
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: skaffoldNamespaceOption}},
			}, &label.DefaultLabeller{}, &test.kubectl)
			t.RequireNoError(err)

			err = k.Deploy(context.Background(), ioutil.Discard, test.builds)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	tests := []struct {
		description string
		kubectl     latestV1.KubectlDeploy
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup success (kubectl v1.18)",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run=client -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup error",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "additional flags",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"deployment.yaml"},
				Flags: latestV1.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"ignored"},
					Delete: []string{"--grace-period=1"},
				},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create -v=0 --dry-run -oyaml -f deployment.yaml", DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete -v=0 --grace-period=1 --ignore-not-found=true --wait=false -f -"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.NewTempDir().
				Write("deployment.yaml", DeploymentWebYAML).
				Chdir()

			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{Namespace: TestNamespace}},
			}, &label.DefaultLabeller{}, &test.kubectl)
			t.RequireNoError(err)

			err = k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlDeployerRemoteCleanup(t *testing.T) {
	tests := []struct {
		description string
		kubectl     latestV1.KubectlDeploy
		commands    util.Command
	}{
		{
			description: "cleanup success",
			kubectl: latestV1.KubectlDeploy{
				RemoteManifests: []string{"pod/leeroy-web"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace get pod/leeroy-web -o yaml").
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -").
				AndRunInput("kubectl --context kubecontext --namespace testNamespace apply -f -", DeploymentWebYAML),
		},
		{
			description: "cleanup error",
			kubectl: latestV1.KubectlDeploy{
				RemoteManifests: []string{"anotherNamespace:pod/leeroy-web"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace anotherNamespace get pod/leeroy-web -o yaml").
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -").
				AndRunInput("kubectl --context kubecontext --namespace anotherNamespace apply -f -", DeploymentWebYAML),
		},
	}
	for _, test := range tests {
		testutil.Run(t, "cleanup remote", func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.NewTempDir().
				Write("deployment.yaml", DeploymentWebYAML).
				Chdir()

			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{Namespace: TestNamespace}},
			}, &label.DefaultLabeller{}, &test.kubectl)
			t.RequireNoError(err)

			err = k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckNoError(err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&client.Client, deployutil.MockK8sClient)
		tmpDir := t.NewTempDir().
			Write("deployment-web.yaml", DeploymentWebYAML).
			Write("deployment-app.yaml", DeploymentAppYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
			AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), DeploymentAppYAML+"\n"+DeploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentAppYAMLv1+"\n---\n"+DeploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext apply -f -", DeploymentAppYAMLv1+"\n---\n"+DeploymentWebYAMLv1).
			AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), DeploymentAppYAML+"\n"+DeploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentAppYAMLv2+"\n---\n"+DeploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext apply -f -", DeploymentAppYAMLv2).
			AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment-app.yaml")+" -f "+tmpDir.Path("deployment-web.yaml"), DeploymentAppYAML+"\n"+DeploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentAppYAMLv2+"\n---\n"+DeploymentWebYAMLv1, ""),
		)

		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: ".",
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   0 * time.Millisecond,
				Max:     10 * time.Second},
		}, &label.DefaultLabeller{}, &latestV1.KubectlDeploy{Manifests: []string{tmpDir.Path("deployment-app.yaml"), tmpDir.Path("deployment-web.yaml")}})
		t.RequireNoError(err)

		// Deploy one manifest
		err = deployer.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
		})
		t.CheckNoError(err)

		// Deploy one manifest since only one image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		})
		t.CheckNoError(err)

		// Deploy zero manifest since no image is updated
		err = deployer.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		})
		t.CheckNoError(err)
	})
}

func TestKubectlWaitForDeletions(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&client.Client, deployutil.MockK8sClient)
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", DeploymentWebYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
			AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment-web.yaml"), DeploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext apply -f -", DeploymentWebYAMLv1),
		)

		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: tmpDir.Root(),
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   0 * time.Millisecond,
				Max:     10 * time.Second,
			},
		}, &label.DefaultLabeller{}, &latestV1.KubectlDeploy{Manifests: []string{tmpDir.Path("deployment-web.yaml")}})
		t.RequireNoError(err)

		var out bytes.Buffer
		err = deployer.Deploy(context.Background(), &out, []graph.Artifact{
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
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", DeploymentWebYAML)

		t.Override(&client.Client, deployutil.MockK8sClient)
		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
			AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment-web.yaml"), DeploymentWebYAML).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}}
				]
			}`),
		)

		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: tmpDir.Root(),
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   10 * time.Second,
				Max:     100 * time.Millisecond,
			},
		}, &label.DefaultLabeller{}, &latestV1.KubectlDeploy{Manifests: []string{tmpDir.Path("deployment-web.yaml")}})
		t.RequireNoError(err)

		err = deployer.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
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

			k, err := NewDeployer(&kubectlConfig{}, &label.DefaultLabeller{}, &latestV1.KubectlDeploy{Manifests: test.manifests})
			t.RequireNoError(err)

			dependencies, err := k.Dependencies()

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, dependencies)
		})
	}
}

func TestKubectlRender(t *testing.T) {
	tests := []struct {
		description string
		builds      []graph.Artifact
		input       string
		expected    string
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
			tmpDir := t.NewTempDir().Write("deployment.yaml", test.input)
			t.Override(&util.DefaultExecCommand, testutil.
				CmdRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext create --dry-run -oyaml -f "+tmpDir.Path("deployment.yaml"), test.input))
			deployer, err := NewDeployer(&kubectlConfig{
				workingDir:  ".",
				defaultRepo: "gcr.io/project",
			}, &label.DefaultLabeller{}, &latestV1.KubectlDeploy{
				Manifests: []string{tmpDir.Path("deployment.yaml")},
			})
			t.RequireNoError(err)
			var b bytes.Buffer
			err = deployer.Render(context.Background(), &b, test.builds, true, "")
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, b.String())
		})
	}
}

func TestGCSManifests(t *testing.T) {
	tests := []struct {
		description string
		kubectl     latestV1.KubectlDeploy
		commands    util.Command
		shouldErr   bool
		skipRender  bool
	}{
		{
			description: "manifest from GCS",
			kubectl: latestV1.KubectlDeploy{
				Manifests: []string{"gs://dev/deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut(fmt.Sprintf("gsutil cp -r %s %s", "gs://dev/deployment.yaml", manifest.ManifestTmpDir), "log").
				AndRunOut("kubectl version --client -ojson", KubectlVersion112).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace create --dry-run -oyaml -f "+filepath.Join(manifest.ManifestTmpDir, "deployment.yaml"), DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
			skipRender: true,
		}}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&client.Client, deployutil.MockK8sClient)
			t.Override(&util.DefaultExecCommand, test.commands)
			if err := os.MkdirAll(manifest.ManifestTmpDir, os.ModePerm); err != nil {
				t.Fatal(err)
			}
			if err := ioutil.WriteFile(manifest.ManifestTmpDir+"/deployment.yaml", []byte(DeploymentWebYAML), os.ModePerm); err != nil {
				t.Fatal(err)
			}
			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				skipRender: test.skipRender,
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{Namespace: TestNamespace}},
			}, &label.DefaultLabeller{}, &test.kubectl)
			t.RequireNoError(err)

			err = k.Deploy(context.Background(), ioutil.Discard, nil)

			t.CheckError(test.shouldErr, err)
		})
	}
}

type kubectlConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	defaultRepo           string
	skipRender            bool
	force                 bool
	waitForDeletions      config.WaitForDeletions
}

func (c *kubectlConfig) GetKubeContext() string                                { return "kubecontext" }
func (c *kubectlConfig) GetKubeNamespace() string                              { return c.Opts.Namespace }
func (c *kubectlConfig) WorkingDir() string                                    { return c.workingDir }
func (c *kubectlConfig) SkipRender() bool                                      { return c.skipRender }
func (c *kubectlConfig) ForceDeploy() bool                                     { return c.force }
func (c *kubectlConfig) DefaultRepo() *string                                  { return &c.defaultRepo }
func (c *kubectlConfig) WaitForDeletions() config.WaitForDeletions             { return c.waitForDeletions }
func (c *kubectlConfig) PortForwardResources() []*latestV1.PortForwardResource { return nil }
