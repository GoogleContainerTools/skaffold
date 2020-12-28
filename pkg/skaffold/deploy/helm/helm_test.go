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

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testBuilds = []build.Artifact{{
	ImageName: "skaffold-helm",
	Tag:       "docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
}}

var testBuildsFoo = []build.Artifact{{
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
			"missing.key": "{{.MISSING}}",
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
		Name:                  "skaffold-helm-remote",
		ChartPath:             "stable/chartmuseum",
		SkipBuildDependencies: false,
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
		Name: "skaffold-helm",
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
	version31   = `version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}`
	version32   = `version.BuildInfo{Version:"v3.2.0", GitCommit:"e11b7ce3b12db2941e90399e874513fbd24bcb71", GitTreeState:"clean", GoVersion:"go1.14"}`
)

func TestBinVer(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
		expected    string
		shouldErr   bool
	}{
		{"Helm 2.0RC1", version20rc, "2.0.0-rc.1", false},
		{"Helm 2.15.1", version21, "2.15.1", false},
		{"Helm 3.0b3", version30b, "3.0.0-beta.3", false},
		{"Helm 3.0", version30, "3.0.0", false},
		{"Helm 3.1.1", version31, "3.1.1", false},
		{"Custom Helm 3.3 build from Manjaro", "v3.3", "3.3.0", false}, // not semver compliant
		{"Invalid", "3.1.0", "0.0.0", true},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))
			ver, err := binVer()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, ver.String())
		})
	}
}

func TestNewDeployer(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
		shouldErr   bool
	}{
		{"Helm 2.0RC1", version20rc, true},
		{"Helm 2.15.1", version21, true},
		{"Helm 3.0.0-beta.0", version30b, false},
		{"Helm 3.0", version30, false},
		{"Helm 3.1.1", version31, false},
		{"helm3 unparseable version", "gobbledygook", true},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))

			_, err := NewDeployer(&helmConfig{}, nil, &testDeployConfig)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHelmDeploy(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "TestHelmDeploy")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	home, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Cannot get homedir: %v", err)
	}

	tests := []struct {
		description        string
		commands           util.Command
		env                []string
		helm               latest.HelmDeploy
		namespace          string
		configure          func(*Deployer)
		builds             []build.Artifact
		force              bool
		shouldErr          bool
		expectedWarnings   []string
		envs               map[string]string
		expectedNamespaces []string
	}{

		{
			description: "helm3.0beta deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30b).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3.0beta namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30b).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3.0 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3.0 namespaced deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3.0 namespaced (with env template) deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseFOOBARNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig", helmReleaseInfo("testReleaseFOOBARNamespace", validDeployYaml)), // just need a valid KRM object
			helm:               testDeployEnvTemplateNamespacedConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseFOOBARNamespace"},
		},
		{
			description: "helm3.0 namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3.0 namespaced context deploy success overrides release namespaces",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployNamespacedConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3.1 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3.1 namespaced deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3.1 namespaced deploy (with env template) success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseFOOBARNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig", helmReleaseInfo("testReleaseFOOBARNamespace", validDeployYaml)), // just need a valid KRM object
			helm:               testDeployEnvTemplateNamespacedConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseFOOBARNamespace"},
		},
		{
			description: "helm3.1 namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3.1 namespaced context deploy success overrides release namespaces",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployNamespacedConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "deploy success with recreatePods",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --recreate-pods examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployRecreatePodsConfig,
			builds: testBuilds,
		},
		{
			description: "deploy success with skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeploySkipBuildDependenciesConfig,
			builds: testBuilds,
		},
		{
			description: "deploy should error for unmatched parameter",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployConfigParameterUnmatched,
			builds:    testBuilds,
			shouldErr: true,
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm stable/chartmuseum --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeploySkipBuildDependencies,
			builds: testBuilds,
		},
		{
			description: "deploy success when `upgradeOnChange: false` and does not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm-upgradeOnChange --kubeconfig kubeconfig"),
			helm: testDeployUpgradeOnChange,
		},
		{
			description: "deploy error remote chart without skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build stable/chartmuseum --kubeconfig kubeconfig", fmt.Errorf("building helm dependencies")),
			helm:      testDeployRemoteChart,
			builds:    testBuilds,
			shouldErr: true,
		},
		{
			description: "get failure should install not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 get failure should install not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "get failure should install not upgrade with helm image strategy",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.repository=docker.io:5000/skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployHelmStyleConfig,
			builds: testBuilds,
		},
		{
			description: "helm image strategy with explicit registry should set the Helm registry value",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.registry=docker.io:5000,image.repository=skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployHelmExplicitRegistryStyleConfig,
			builds: testBuilds,
		},
		{
			description: "get success should upgrade by force, not install",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --force examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			force:  true,
			builds: testBuilds,
		},
		{
			description: "get success should upgrade without force, not install",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "deploy error",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			shouldErr: true,
			helm:      testDeployConfig,
			builds:    testBuilds,
		},
		{
			description: "dep build error",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			shouldErr: true,
			helm:      testDeployConfig,
			builds:    testBuilds,
		},
		{
			description: "helm 3.0 beta should package chart and deploy",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30b).
				AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Sprintf("Packaged to %s", filepath.Join(tmpDir, "foo-0.1.2.tgz"))).
				AndRun("helm --kube-context kubecontext upgrade foo " + filepath.Join(tmpDir, "foo-0.1.2.tgz") + " --set-string image=foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig"),
			shouldErr: false,
			helm:      testDeployFooWithPackaged,
			builds:    testBuildsFoo,
		},
		{
			description: "helm 3.1 should package chart and deploy",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Sprintf("Packaged to %s", filepath.Join(tmpDir, "foo-0.1.2.tgz"))).
				AndRun("helm --kube-context kubecontext upgrade foo " + filepath.Join(tmpDir, "foo-0.1.2.tgz") + " --set-string image=foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig"),
			shouldErr: false,
			helm:      testDeployFooWithPackaged,
			builds:    testBuildsFoo,
		},
		{
			description: "should fail to deploy when packaging fails",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Errorf("packaging failed")),
			shouldErr: true,
			helm:      testDeployFooWithPackaged,
			builds:    testBuildsFoo,
		},
		{
			description: "deploy and get missing templated release name should fail",
			commands:    testutil.CmdRunWithOutput("helm version --client", version31),
			helm:        testDeployWithTemplatedName,
			builds:      testBuilds,
			shouldErr:   true,
		},
		{
			description: "deploy and get templated release name",
			env:         []string{"USER=user"},
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all user-skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade user-skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all user-skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployWithTemplatedName,
			builds: testBuilds,
		},
		{
			description: "deploy with templated values",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set image.name=skaffold-helm --set image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set missing.key=<no value> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfigTemplated,
			builds: testBuilds,
		},
		{
			description: "deploy with valuesFiles templated",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfigValuesFilesTemplated,
			builds: testBuilds,
		},
		{
			description: "deploy with setFiles",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun(fmt.Sprintf("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set-file expanded=%s --set-file value=/some/file.yaml --kubeconfig kubeconfig", filepath.Join(home, "file.yaml"))).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfigSetFiles,
			builds: testBuilds,
		},
		{
			description: "deploy without actual tags",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployWithoutTags,
			builds: testBuilds,
			expectedWarnings: []string{
				"See helm sample for how to replace image names with their actual tags: https://github.com/GoogleContainerTools/skaffold/blob/master/examples/helm-deployment/skaffold.yaml",
				"image [docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184] is not used.",
				"image [skaffold-helm] is used instead.",
			},
		},
		{
			description: "first release without tag, second with tag",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade other examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build  --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm  --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:   testTwoReleases,
			builds: testBuilds,
		},
		{
			description: "debug for helm3.0 failure",
			commands:    testutil.CmdRunWithOutput("helm version --client", version30),
			shouldErr:   true,
			helm:        testDeployConfig,
			builds:      testBuilds,
			configure:   func(deployer *Deployer) { deployer.enableDebug = true },
		},
		{
			description: "debug for helm3.1 success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm --post-renderer SKAFFOLD-BINARY examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml"}).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			builds:    testBuilds,
			configure: func(deployer *Deployer) { deployer.enableDebug = true },
		},
		{
			description: "helm3.1 should fail to deploy with createNamespace option",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig"),
			helm:      testDeployCreateNamespaceConfig,
			builds:    testBuilds,
			shouldErr: true,
		},
		{
			description: "helm3.2 get failure should install with createNamespace not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version32).
				AndRunErr("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test --namespace testReleaseNamespace --create-namespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", helmReleaseInfo("testReleaseNamespace", validDeployYaml)), // just need a valid KRM object
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
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig", helmReleaseInfo("testReleaseNamespace", validDeployYaml)), // just need a valid KRM object
			helm:               testDeployCreateNamespaceConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseNamespace"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			env := test.env
			if env == nil {
				env = []string{"FOO=FOOBAR"}
			}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return env })
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&osExecutable, func() (string, error) { return "SKAFFOLD-BINARY", nil })

			deployer, err := NewDeployer(&helmConfig{
				namespace:  test.namespace,
				force:      test.force,
				configFile: "test.yaml",
			}, nil, &test.helm)
			t.RequireNoError(err)

			if test.configure != nil {
				test.configure(deployer)
			}
			deployer.pkgTmpDir = tmpDir
			// Deploy returns nil unless `helm get all <release>` is set up to return actual release info
			nss, err := deployer.Deploy(context.Background(), ioutil.Discard, test.builds)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedNamespaces, nss)
		})
	}
}

func TestHelmCleanup(t *testing.T) {
	tests := []struct {
		description      string
		commands         util.Command
		helm             latest.HelmDeploy
		namespace        string
		builds           []build.Artifact
		expectedWarnings []string
		envs             map[string]string
	}{
		{
			description: "helm3 cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespace cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			helm:   testDeployNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespace (with env template) cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testReleaseFOOBARNamespace --kubeconfig kubeconfig"),
			helm:   testDeployEnvTemplateNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success overriding release namespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			helm:      testDeployNamespacedConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=FOOBAR"} })
			t.Override(&util.DefaultExecCommand, test.commands)

			deployer, err := NewDeployer(&helmConfig{
				namespace: test.namespace,
			}, nil, &test.helm)
			t.RequireNoError(err)

			deployer.Cleanup(context.Background(), ioutil.Discard)

			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}

func TestParseHelmRelease(t *testing.T) {
	tests := []struct {
		description string
		yaml        []byte
		shouldErr   bool
	}{
		{
			description: "parse valid deployment yaml",
			yaml:        []byte(validDeployYaml),
		},
		{
			description: "parse valid service yaml",
			yaml:        []byte(validServiceYaml),
		},
		{
			description: "parse invalid deployment yaml",
			yaml:        []byte(invalidDeployYaml),
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := parseRuntimeObject(kubectl.TestNamespace, test.yaml)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHelmDependencies(t *testing.T) {
	tests := []struct {
		description           string
		files                 []string
		valuesFiles           []string
		skipBuildDependencies bool
		remote                bool
		expected              func(folder *testutil.TempDir) []string
	}{
		{
			description:           "charts download dir and lock files are included when skipBuildDependencies is true",
			files:                 []string{"Chart.yaml", "Chart.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: true,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.lock"),
					folder.Path("Chart.yaml"),
					folder.Path("charts/xyz.tar"),
					folder.Path("templates/deploy.yaml"),
					folder.Path("tmpcharts/xyz.tar"),
				}
			},
		},
		{
			description:           "charts download dir and lock files are excluded when skipBuildDependencies is false",
			files:                 []string{"Chart.yaml", "Chart.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: false,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.yaml"),
					folder.Path("templates/deploy.yaml"),
				}
			},
		},
		{
			description:           "values file is included",
			skipBuildDependencies: false,
			files:                 []string{"Chart.yaml"},
			valuesFiles:           []string{"/folder/values.yaml"},
			expected: func(folder *testutil.TempDir) []string {
				return []string{"/folder/values.yaml", folder.Path("Chart.yaml")}
			},
		},
		{
			description:           "no deps for remote chart path",
			skipBuildDependencies: false,
			files:                 []string{"Chart.yaml"},
			remote:                true,
			expected: func(folder *testutil.TempDir) []string {
				return nil
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version30))

			tmpDir := t.NewTempDir().
				Touch(test.files...)

			deployer, err := NewDeployer(&helmConfig{}, nil, &latest.HelmDeploy{
				Releases: []latest.HelmRelease{{
					Name:                  "skaffold-helm",
					ChartPath:             tmpDir.Root(),
					ValuesFiles:           test.valuesFiles,
					ArtifactOverrides:     map[string]string{"image": "skaffold-helm"},
					Overrides:             schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
					SetValues:             map[string]string{"some.key": "somevalue"},
					SkipBuildDependencies: test.skipBuildDependencies,
					Remote:                test.remote,
				}},
			})
			t.RequireNoError(err)
			deps, err := deployer.Dependencies()

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected(tmpDir), deps)
		})
	}
}

func TestImageSetFromConfig(t *testing.T) {
	tests := []struct {
		description string
		valueName   string
		tag         string
		expected    string
		strategy    *latest.HelmConventionConfig
		shouldErr   bool
	}{
		{
			description: "Helm set values with no convention config",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "image=skaffold-helm:1.0.0",
			strategy:    nil,
			shouldErr:   false,
		},
		{
			description: "Helm set values with helm conventions",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "image.repository=skaffold-helm,image.tag=1.0.0",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   false,
		},
		{
			description: "Helm set values with helm conventions and explicit registry value",
			valueName:   "image",
			tag:         "docker.io/skaffold-helm:1.0.0",
			expected:    "image.registry=docker.io,image.repository=skaffold-helm,image.tag=1.0.0",
			strategy: &latest.HelmConventionConfig{
				ExplicitRegistry: true,
			},
			shouldErr: false,
		},
		{
			description: "Invalid tag with helm conventions",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0,0",
			expected:    "",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   true,
		},
		{
			description: "Helm set values with helm conventions and explicit registry value, but missing in tag",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "",
			strategy: &latest.HelmConventionConfig{
				ExplicitRegistry: true,
			},
			shouldErr: true,
		},
		{
			description: "Helm set values using digest",
			valueName:   "image",
			tag:         "skaffold-helm:stable@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			expected:    "image.repository=skaffold-helm,image.tag=stable@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			values, err := imageSetFromConfig(test.strategy, test.valueName, test.tag)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, values)
		})
	}
}

func TestHelmRender(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		commands    util.Command
		helm        latest.HelmDeploy
		outputFile  string
		expected    string
		builds      []build.Artifact
		envs        map[string]string
		namespace   string
	}{
		{
			description: "normal render v3",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfig,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render to a file",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunWithOutput("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig",
					"Dummy Output"),
			helm:       testDeployConfig,
			outputFile: "dummy.yaml",
			expected:   "Dummy Output\n",
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated config",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set image.name=skaffold-helm --set image.tag=skaffold-helm:tag1 --set missing.key=<no value> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfigTemplated,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated values file",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig"),
			helm: testDeployConfigValuesFilesTemplated,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			helm: testDeployNamespacedConfig,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace testReleaseFOOBARNamespace --kubeconfig kubeconfig"),
			helm: testDeployEnvTemplateNamespacedConfig,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with cli namespace",
			shouldErr:   false,
			namespace:   "clinamespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
			helm: testDeployConfig,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with HelmRelease.Namespace and cli namespace",
			shouldErr:   false,
			namespace:   "clinamespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
			helm: testDeployNamespacedConfig,
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file := ""
			if test.outputFile != "" {
				file = t.NewTempDir().Path(test.outputFile)
			}

			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=FOOBAR"} })
			t.Override(&util.DefaultExecCommand, test.commands)
			deployer, err := NewDeployer(&helmConfig{
				namespace: test.namespace,
			}, nil, &test.helm)
			t.RequireNoError(err)
			err = deployer.Render(context.Background(), ioutil.Discard, test.builds, true, file)
			t.CheckError(test.shouldErr, err)

			if file != "" {
				dat, _ := ioutil.ReadFile(file)
				t.CheckDeepEqual(string(dat), test.expected)
			}
		})
	}
}

func TestWriteBuildArtifacts(t *testing.T) {
	tests := []struct {
		description string
		builds      []build.Artifact
		result      string
	}{
		{
			description: "nil",
			builds:      nil,
			result:      `{"builds":null}`,
		},
		{
			description: "empty",
			builds:      []build.Artifact{},
			result:      `{"builds":[]}`,
		},
		{
			description: "multiple images with tags",
			builds:      []build.Artifact{{ImageName: "name", Tag: "name:tag"}, {ImageName: "name2", Tag: "name2:tag"}},
			result:      `{"builds":[{"imageName":"name","tag":"name:tag"},{"imageName":"name2","tag":"name2:tag"}]}`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file, cleanup, err := writeBuildArtifacts(test.builds)
			t.CheckError(false, err)
			if content, err := ioutil.ReadFile(file); err != nil {
				t.Errorf("error reading file %q: %v", file, err)
			} else {
				t.CheckDeepEqual(test.result, string(content))
			}
			cleanup()
		})
	}
}

func TestGenerateSkaffoldDebugFilter(t *testing.T) {
	tests := []struct {
		description string
		buildFile   string
		result      []string
	}{
		{
			description: "empty buildfile is skipped",
			buildFile:   "",
			result:      []string{"filter", "--debugging", "--kube-context", "kubecontext", "--kubeconfig", "kubeconfig"},
		},
		{
			description: "buildfile is added",
			buildFile:   "buildfile",
			result:      []string{"filter", "--debugging", "--kube-context", "kubecontext", "--build-artifacts", "buildfile", "--kubeconfig", "kubeconfig"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			h, err := NewDeployer(&helmConfig{}, nil, &testDeployConfig)
			t.RequireNoError(err)
			result := h.generateSkaffoldDebugFilter(test.buildFile)
			t.CheckDeepEqual(test.result, result)
		})
	}
}

type helmConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	namespace             string
	force                 bool
	configFile            string
}

func (c *helmConfig) ForceDeploy() bool         { return c.force }
func (c *helmConfig) GetKubeConfig() string     { return kubectl.TestKubeConfig }
func (c *helmConfig) GetKubeContext() string    { return kubectl.TestKubeContext }
func (c *helmConfig) GetKubeNamespace() string  { return c.namespace }
func (c *helmConfig) ConfigurationFile() string { return c.configFile }

// helmReleaseInfo returns the result of `helm --namespace <namespace> get all <name>` with the given KRM manifest.
func helmReleaseInfo(namespace, manifest string) string {
	return fmt.Sprintf(`NAME: skaffold-helm
LAST DEPLOYED: Thu Dec 17 15:35:28 2020
NAMESPACE: %s
STATUS: deployed
REVISION: 1
TEST SUITE: None
USER-SUPPLIED VALUES:
image: skaffold-example:16bb174b9a147d3f574fb5fe967b7f5c873a0150182dbb0f72d1fb2fffd69a12

COMPUTED VALUES:
image: skaffold-example:16bb174b9a147d3f574fb5fe967b7f5c873a0150182dbb0f72d1fb2fffd69a12

HOOKS:
MANIFEST:
---
%s`, namespace, manifest)
}
