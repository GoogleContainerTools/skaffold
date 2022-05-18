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
	"io/ioutil"
	"testing"

	"github.com/blang/semver"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	// Output strings to emulate different versions of Helm
	version31 = `version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}`
	version32 = `version.BuildInfo{Version:"v3.2.0", GitCommit:"e11b7ce3b12db2941e90399e874513fbd24bcb71", GitTreeState:"clean", GoVersion:"go1.14"}`
	version35 = `version.BuildInfo{Version:"3.5.2", GitCommit:"c4e74854886b2efe3321e185578e6db9be0a6e29", GitTreeState:"clean", GoVersion:"go1.14.15"}`
)

func TestNewDeployer31(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
	}{
		{"Helm 3.1.1", version31},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))

			_, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig)
			t.CheckNoError(err)
		})
	}
}

func TestHelmDeploy31(t *testing.T) {
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
		configure          func(*Deployer31)
		builds             []graph.Artifact
		force              bool
		shouldErr          bool
		expectedWarnings   []string
		expectedNamespaces []string
	}{
		{
			description:        "helm3.1 deploy fails",
			helm:               testDeployConfig,
			builds:             testBuilds,
			shouldErr:          true,
			expectedNamespaces: []string{""},
		},
		/*
			{
				description: "helm3.1 deploy success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm3.1 namespaced deploy success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployNamespacedConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{"testReleaseNamespace"},
			},
			{
				description: "helm3.1 namespaced deploy (with env template) success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseFOOBARNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployEnvTemplateNamespacedConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{"testReleaseFOOBARNamespace"},
			},
			{
				description: "helm3.1 deploy with repo success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --repo https://charts.helm.sh/stable --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfigRemoteRepo,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm3.1 namespaced context deploy success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				namespace:          kubectl.TestNamespace,
				builds:             testBuilds,
				expectedNamespaces: []string{"testNamespace"},
			},
			{
				description: "helm3.1 namespaced context deploy success overrides release namespaces",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployNamespacedConfig,
				namespace:          kubectl.TestNamespace,
				builds:             testBuilds,
				expectedNamespaces: []string{"testNamespace"},
			},
			{
				description: "deploy success with recreatePods",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm --recreate-pods examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployRecreatePodsConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy success with skipBuildDependencies",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeploySkipBuildDependenciesConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy should error for unmatched parameter",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfigParameterUnmatched,
				builds:             testBuilds,
				shouldErr:          true,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy success remote chart with skipBuildDependencies",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm stable/chartmuseum --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeploySkipBuildDependencies,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy success when `upgradeOnChange: false` and does not upgrade",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm-upgradeOnChange --kubeconfig kubeconfig"),
				helm:               testDeployUpgradeOnChange,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy remote chart",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig", fmt.Errorf("Error: release: not found")).
					AndRun("helm --kube-context kubecontext install skaffold-helm-remote stable/chartmuseum --repo https://charts.helm.sh/stable --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm-remote --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployRemoteChart,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy remote chart with version",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig", fmt.Errorf("Error: release: not found")).
					AndRun("helm --kube-context kubecontext install skaffold-helm-remote --version 1.0.0 stable/chartmuseum --repo https://charts.helm.sh/stable --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm-remote --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployRemoteChartVersion,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy error with remote chart",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig", fmt.Errorf("Error: release: not found")).
					AndRunErr("helm --kube-context kubecontext install skaffold-helm-remote stable/chartmuseum --repo https://charts.helm.sh/stable --kubeconfig kubeconfig", fmt.Errorf("building helm dependencies")),
				helm:               testDeployRemoteChart,
				shouldErr:          true,
				expectedNamespaces: []string{""},
			},
			{
				description: "get failure should install not upgrade",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm3 get failure should install not upgrade",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "get failure should install not upgrade with helm image strategy",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --set-string image.repository=docker.io:5000/skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployHelmStyleConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm image strategy with explicit registry should set the Helm registry value",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --set-string image.registry=docker.io:5000,image.repository=skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployHelmExplicitRegistryStyleConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "get success should upgrade by force, not install",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm --force examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				force:              true,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "get success should upgrade without force, not install",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy error",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRunErr("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				shouldErr:          true,
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "dep build error",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRunErr("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				shouldErr:          true,
				helm:               testDeployConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm 3.1 should package chart and deploy",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
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
				description: "should fail to deploy when packaging fails",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
					AndRunErr("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Errorf("packaging failed")),
				shouldErr:          true,
				helm:               testDeployFooWithPackaged,
				builds:             testBuildsFoo,
				expectedNamespaces: []string{""},
			},
			{
				description:        "deploy and get missing templated release name should fail",
				commands:           testutil.CmdRunWithOutput("helm version --client", version31),
				helm:               testDeployWithTemplatedName,
				builds:             testBuilds,
				shouldErr:          true,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy and get templated release name",
				env:         []string{"USER=user"},
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all user-skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade user-skaffold-helm examples/test --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all user-skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployWithTemplatedName,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy with templated values",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set image.name=skaffold-helm --set image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set missing.key=<MISSING> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfigTemplated,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy with valuesFiles templated",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 -f /some/file-FOOBAR.yaml -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfigValuesFilesTemplated,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy with templated version",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm --version 1.0 examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				env:                []string{"VERSION=1.0"},
				helm:               testDeployConfigVersionTemplated,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy with setFiles",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun(fmt.Sprintf("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set-file expanded=%s --set-file value=/some/file.yaml -f skaffold-overrides.yaml --kubeconfig kubeconfig", strings.ReplaceAll(filepath.Join(home, "file.yaml"), "\\", "\\\\"))).
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfigSetFiles,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "deploy without actual tags",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:   testDeployWithoutTags,
				builds: testBuilds,
				expectedWarnings: []string{
					"See helm documentation on how to replace image names with their actual tags: https://skaffold.dev/docs/pipeline-stages/deployers/helm/#image-configuration",
					"image [docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184] is not used.",
				},
				expectedNamespaces: []string{""},
			},
			{
				description: "first release without tag, second with tag",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all other --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade other examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext get all other --template {{.Release.Manifest}} --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testTwoReleases,
				builds:             testBuilds,
				expectedNamespaces: []string{""},
			},
			{
				description: "debug for helm3.1 success",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm --post-renderer SKAFFOLD-BINARY examples/test --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
						[]string{"SKAFFOLD_FILENAME=test.yaml"}).
					AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployConfig,
				builds:             testBuilds,
				configure:          func(deployer *Deployer31) { deployer.enableDebug = true },
				expectedNamespaces: []string{""},
			},
			{
				description: "helm3.1 should fail to deploy with createNamespace option",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version31).
					AndRunErr("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig"),
				helm:               testDeployCreateNamespaceConfig,
				builds:             testBuilds,
				shouldErr:          true,
				expectedNamespaces: []string{""},
			},
			{
				description: "helm3.2 get failure should install with createNamespace not upgrade",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version32).
					AndRunErr("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --namespace testReleaseNamespace --create-namespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployCreateNamespaceConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{"testReleaseNamespace"},
			},
			{
				description: "helm3.2 namespaced deploy success without createNamespace",
				commands: testutil.
					CmdRunWithOutput("helm version --client", version32).
					AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
					AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig").
					AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
				helm:               testDeployCreateNamespaceConfig,
				builds:             testBuilds,
				expectedNamespaces: []string{"testReleaseNamespace"},
			}*/
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

			v, err := semver.New("3.1.0")
			t.RequireNoError(err)

			deployer, err := NewDeployer31(context.Background(), &helmConfig{
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
