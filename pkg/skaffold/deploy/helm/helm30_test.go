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

package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/blang/semver"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHelmDeploy30(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "TestHelmDeploy")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}

	tests := []struct {
		description        string
		commands           util.Command
		env                []string
		helm               latest.HelmDeploy
		namespace          string
		configure          func(*Deployer30)
		builds             []graph.Artifact
		force              bool
		shouldErr          bool
		expectedWarnings   []string
		expectedNamespaces []string
	}{
		{
			description: "helm3.0beta deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "helm3.0beta namespaced context deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			namespace:          kubectl.TestNamespace,
			builds:             testBuilds,
			expectedNamespaces: []string{"testNamespace"},
		},
		{
			description: "helm3.0 deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "helm3.0 namespaced deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployNamespacedConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseNamespace"},
		},
		{
			description: "helm3.0 namespaced (with env template) deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseFOOBARNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployEnvTemplateNamespacedConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseFOOBARNamespace"},
		},
		{
			description: "helm3.0 namespaced context deploy success",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			namespace:          kubectl.TestNamespace,
			builds:             testBuilds,
			expectedNamespaces: []string{"testNamespace"},
		},
		{
			description: "helm3.0 namespaced context deploy success overrides release namespaces",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployNamespacedConfig,
			namespace:          kubectl.TestNamespace,
			builds:             testBuilds,
			expectedNamespaces: []string{"testNamespace"},
		},
		{
			description: "helm 3.0 beta should package chart and deploy",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Sprintf("Packaged to %s", filepath.Join(tmpDir, "foo-0.1.2.tgz"))).
				AndRun("helm --kube-context kubecontext upgrade foo "+filepath.Join(tmpDir, "foo-0.1.2.tgz")+" --set-string image=foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all foo --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			shouldErr:          false,
			helm:               testDeployFooWithPackaged,
			builds:             testBuildsFoo,
			expectedNamespaces: []string{""},
		},
		{
			description:        "debug for helm3.0 failure",
			shouldErr:          true,
			helm:               testDeployConfig,
			builds:             testBuilds,
			configure:          func(deployer *Deployer30) { deployer.enableDebug = true },
			expectedNamespaces: []string{""},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&client.Client, deployutil.MockK8sClient)
			fakeWarner := &warnings.Collect{}
			env := test.env
			if env == nil {
				env = []string{"FOO=FOOBAR"}
			}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return env })
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&osExecutable, func() (string, error) { return "SKAFFOLD-BINARY", nil })
			t.Override(&kubectx.CurrentConfig, func() (api.Config, error) {
				return api.Config{CurrentContext: ""}, nil
			})

			v, err := semver.New("3.0.0")
			t.RequireNoError(err)
			deployer, err := NewDeployer30(context.Background(), &helmConfig{
				namespace:  test.namespace,
				force:      test.force,
				configFile: "test.yaml",
			}, &label.DefaultLabeller{}, &test.helm, *v)

			t.RequireNoError(err)

			if test.configure != nil {
				test.configure(deployer)
			}
			deployer.pkgTmpDir = tmpDir
			// Deploy returns nil unless `helm get all <release>` is set up to return actual release info
			err = deployer.Deploy(context.Background(), ioutil.Discard, test.builds)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedNamespaces, *deployer.namespaces)
		})
	}
}
