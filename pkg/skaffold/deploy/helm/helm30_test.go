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

	"github.com/blang/semver"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testBuilds = []graph.Artifact{{
	ImageName: "skaffold-helm",
	Tag:       "docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
}}

var testBuildsFoo = []graph.Artifact{{
	ImageName: "foo",
	Tag:       "foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
}}

var testDeployConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
	}},
}

var testDeployNamespacedConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Namespace: "testReleaseNamespace",
	}},
}

var testDeployEnvTemplateNamespacedConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Namespace: "testRelease{{.FOO}}Namespace",
	}},
}

var testDeployConfigRemoteRepo = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Repo: "https://charts.helm.sh/stable",
	}},
}

var testDeployConfigTemplated = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValueTemplates: map[string]string{
			"some.key":    "somevalue",
			"other.key":   "{{.FOO}}",
			"missing.key": `{{default "<MISSING>" .MISSING}}`,
			"image.name":  "{{.IMAGE_NAME}}",
			"image.tag":   "{{.DIGEST}}",
			"{{.FOO}}":    "somevalue",
		},
	}},
}

var testDeployConfigValuesFilesTemplated = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		ValuesFiles: []string{
			"/some/file-{{.FOO}}.yaml",
		},
	}},
}

var testDeployConfigVersionTemplated = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Version: "{{.VERSION}}",
	}},
}

var testDeployConfigSetFiles = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetFiles: map[string]string{
			"expanded": "~/file.yaml",
			"value":    "/some/file.yaml",
		},
	}},
}

var testDeployRecreatePodsConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		RecreatePods: true,
	}},
}

var testDeploySkipBuildDependenciesConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		SkipBuildDependencies: true,
	}},
}

var testDeployHelmStyleConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		ImageStrategy: latest.HelmImageStrategy{
			HelmImageConfig: latest.HelmImageConfig{
				HelmConventionConfig: &latest.HelmConventionConfig{},
			},
		},
	}},
}

var testDeployHelmExplicitRegistryStyleConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		ImageStrategy: latest.HelmImageStrategy{
			HelmImageConfig: latest.HelmImageConfig{
				HelmConventionConfig: &latest.HelmConventionConfig{
					ExplicitRegistry: true,
				},
			},
		},
	}},
}

var testDeployConfigParameterUnmatched = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm-unmatched",
		}},
	},
}

var testDeployFooWithPackaged = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "foo",
		ChartPath: "testdata/foo",
		ArtifactOverrides: map[string]string{
			"image": "foo",
		},
		Packaged: &latest.HelmPackaged{
			Version:    "0.1.2",
			AppVersion: "1.2.3",
		},
	}},
}

var testDeployWithTemplatedName = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "{{.USER}}-skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image.tag": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		}},
	},
}

var testDeploySkipBuildDependencies = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "stable/chartmuseum",
		ArtifactOverrides: map[string]string{
			"image.tag": "skaffold-helm",
		},
		SkipBuildDependencies: true,
	}},
}

var testDeployRemoteChart = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:        "skaffold-helm-remote",
		RemoteChart: "stable/chartmuseum",
		Repo:        "https://charts.helm.sh/stable",
	}},
}

var testDeployRemoteChartVersion = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:        "skaffold-helm-remote",
		RemoteChart: "stable/chartmuseum",
		Version:     "1.0.0",
		Repo:        "https://charts.helm.sh/stable",
	}},
}

var upgradeOnChangeFalse = false
var testDeployUpgradeOnChange = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:            "skaffold-helm-upgradeOnChange",
		ChartPath:       "examples/test",
		UpgradeOnChange: &upgradeOnChangeFalse,
	}},
}

var testDeployWithoutTags = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
	}},
}

var testTwoReleases = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "other",
		ChartPath: "examples/test",
	}, {
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image.tag": "skaffold-helm",
		},
	}},
}

var createNamespaceFlag = true
var testDeployCreateNamespaceConfig = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		ArtifactOverrides: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Namespace:       "testReleaseNamespace",
		CreateNamespace: &createNamespaceFlag,
	}},
}

var validDeployYaml = `
# Source: skaffold-helm/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Tiller
spec:
  replicas: 1
  selector:
    matchLabels:
      app: skaffold-helm
      release: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        release: skaffold-helm
    spec:
      containers:
        - name: skaffold-helm
          image: gcr.io/nick-cloudbuild/skaffold-helm:f759510436c8fd6f7ffa13dd9e9d85e64bec8d2bfd12c5aa3fb9af1288eccdab
          imagePullPolicy:
          command: ["/bin/bash", "-c", "--" ]
          args: ["while true; do sleep 30; done;"]
          resources:
            {}
`

var validServiceYaml = `
# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Tiller
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
      name: nginx
  selector:
    app: skaffold-helm
    release: skaffold-helm
`

var invalidDeployYaml = `REVISION: 2
RELEASED: Tue Jun 12 15:40:18 2018
CHART: skaffold-helm-0.1.0
USER-SUPPLIED VALUES:
image: gcr.io/nick-cloudbuild/skaffold-helm:f759510436c8fd6f7ffa13dd9e9d85e64bec8d2bfd12c5aa3fb9af1288eccdab

COMPUTED VALUES:
image: gcr.io/nick-cloudbuild/skaffold-helm:f759510436c8fd6f7ffa13dd9e9d85e64bec8d2bfd12c5aa3fb9af1288eccdab
ingress:
  annotations: null
  enabled: false
  hosts:
  - chart-example.local
  tls: null
replicaCount: 1
resources: {}
service:
  externalPort: 80
  internalPort: 80
  name: nginx
  type: ClusterIP

HOOKS:
MANIFEST:
`

var (
	// Output strings to emulate different versions of Helm
	version20rc = `Client: &version.Version{SemVer:"v2.0.0-rc.1", GitCommit:"92be174acf51e60a33287fb7011f4571eaa5cb98", GitTreeState:"clean"}\nError: cannot connect to Tiller\n`
	version21   = `Client: &version.Version{SemVer:"v2.15.1", GitCommit:"cf1de4f8ba70eded310918a8af3a96bfe8e7683b", GitTreeState:"clean"}\nServer: &version.Version{SemVer:"v2.16.1", GitCommit:"bbdfe5e7803a12bbdf97e94cd847859890cf4050", GitTreeState:"clean"}\n`
	version30b  = `version.BuildInfo{Version:"v3.0.0-beta.3", GitCommit:"5cb923eecbe80d1ad76399aee234717c11931d9a", GitTreeState:"clean", GoVersion:"go1.12.9"}`
	version30   = `version.BuildInfo{Version:"v3.0.0", GitCommit:"e29ce2a54e96cd02ccfce88bee4f58bb6e2a28b6", GitTreeState:"clean", GoVersion:"go1.13.4"}`
)

func TestNewDeployer30(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
	}{
		{"Helm 3.0.0-beta.0", version30b},
		{"Helm 3.0", version30},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))

			_, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig)
			t.CheckNoError(err)
		})
	}
}

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

type helmConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	namespace             string
	force                 bool
	configFile            string
}

func (c *helmConfig) ForceDeploy() bool                                   { return c.force }
func (c *helmConfig) GetKubeConfig() string                               { return kubectl.TestKubeConfig }
func (c *helmConfig) GetKubeContext() string                              { return kubectl.TestKubeContext }
func (c *helmConfig) GetKubeNamespace() string                            { return c.namespace }
func (c *helmConfig) ConfigurationFile() string                           { return c.configFile }
func (c *helmConfig) PortForwardResources() []*latest.PortForwardResource { return nil }
