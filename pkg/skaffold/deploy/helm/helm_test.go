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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	ctl "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	rhelm "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var testBuilds = []graph.Artifact{{
	ImageName: "skaffold-helm",
	Tag:       "docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
}}

var testDeployConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
	}},
}

var testDeployNamespacedConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Namespace: "testReleaseNamespace",
	}},
}

var testDeployEnvTemplateNamespacedConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Namespace: "testRelease{{.FOO}}Namespace",
	}},
}

var testDeployWithTemplatedChartPath = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/{{.FOO}}",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
	}},
}

var testDeployConfigRemoteRepo = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		Repo: "https://charts.helm.sh/stable",
	}},
}

var testDeployConfigTemplated = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
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

var testDeployConfigValuesFilesTemplated = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		ValuesFiles: []string{
			"/some/file-{{.FOO}}.yaml",
		},
	}},
}

var testDeployConfigVersionTemplated = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Version:   "{{.VERSION}}",
	}},
}
var testDeployConfigRepoTemplated = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Repo:      "https://{{.CHARTUSER}}:{{.CHARTPASS}}@charts.helm.sh/stable",
	}},
}

var testDeployConfigSetFiles = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetFiles: map[string]string{
			"expanded": "~/file.yaml",
			"value":    "/some/file.yaml",
		},
	}},
}

var testDeployRecreatePodsConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		RecreatePods: true,
	}},
}

var testDeploySkipBuildDependenciesConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
		SkipBuildDependencies: true,
	}},
}

var testDeployWithPackaged = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "testdata/skaffold-helm",
		Packaged: &latest.HelmPackaged{
			Version:    "0.1.2",
			AppVersion: "1.2.3",
		},
	}},
}

var testDeployWithTemplatedName = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "{{.USER}}-skaffold-helm",
		ChartPath: "examples/test",
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		}},
	},
}

var testDeploySkipBuildDependencies = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:                  "skaffold-helm",
		ChartPath:             "stable/chartmuseum",
		SkipBuildDependencies: true,
	}},
}

var testDeployRemoteChart = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:        "skaffold-helm-remote",
		RemoteChart: "stable/chartmuseum",
		Repo:        "https://charts.helm.sh/stable",
	}},
}

var testDeployRemoteChartVersion = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:        "skaffold-helm-remote",
		RemoteChart: "stable/chartmuseum",
		Version:     "1.0.0",
		Repo:        "https://charts.helm.sh/stable",
	}},
}

var upgradeOnChangeFalse = false
var testDeployUpgradeOnChange = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:            "skaffold-helm-upgradeOnChange",
		ChartPath:       "examples/test",
		UpgradeOnChange: &upgradeOnChangeFalse,
	}},
}

var testTwoReleases = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "other",
		ChartPath: "examples/test",
	}, {
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
	}},
}

var createNamespaceFlag = true
var testDeployCreateNamespaceConfig = latest.LegacyHelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
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
          image: docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184
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
	version35   = `version.BuildInfo{Version:"3.5.2", GitCommit:"c4e74854886b2efe3321e185578e6db9be0a6e29", GitTreeState:"clean", GoVersion:"go1.14.15"}`
)

func TestNewDeployer(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
		shouldErr   bool
	}{
		{"Helm 2.0RC1", version20rc, true},
		{"Helm 2.15.1", version21, true},
		{"Helm 3.0.0-beta.0", version30b, true},
		{"Helm 3.0", version30, true},
		{"Helm 3.1.0", version31, false},
		{"helm3 unparseable version", "gobbledygook", true},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))

			_, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig, nil, "default")
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHelmDeploy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestHelmDeploy")
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
		helm               latest.LegacyHelmDeploy
		namespace          string
		configure          func(*Deployer)
		builds             []graph.Artifact
		force              bool
		shouldErr          bool
		expectedWarnings   []string
		expectedNamespaces []string
	}{
		{
			description: "helm3.1 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "helm3.1 namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			namespace:          kubectl.TestNamespace,
			builds:             testBuilds,
			expectedNamespaces: []string{"testNamespace"},
		},
		{
			description: "helm3.5 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version35).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployNamespacedConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseNamespace"},
		},
		{
			description: "helm3.1 namespaced (with env template) deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseFOOBARNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseFOOBARNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployNamespacedConfig,
			namespace:          kubectl.TestNamespace,
			builds:             testBuilds,
			expectedNamespaces: []string{"testNamespace"},
		},
		{
			description: "helm3.1 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseFOOBARNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --repo https://charts.helm.sh/stable --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm --recreate-pods examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeploySkipBuildDependenciesConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm stable/chartmuseum --post-renderer SKAFFOLD-BINARY --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext install skaffold-helm-remote stable/chartmuseum --post-renderer SKAFFOLD-BINARY --repo https://charts.helm.sh/stable --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm-remote --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployRemoteChart,
			expectedNamespaces: []string{""},
		},
		{
			description: "deploy remote chart with version",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig", fmt.Errorf("Error: release: not found")).
				AndRunEnv("helm --kube-context kubecontext install skaffold-helm-remote --version 1.0.0 stable/chartmuseum --post-renderer SKAFFOLD-BINARY --repo https://charts.helm.sh/stable --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm-remote --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployRemoteChartVersion,
			expectedNamespaces: []string{""},
		},
		{
			description: "deploy error with remote chart",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm-remote --kubeconfig kubeconfig", fmt.Errorf("Error: release: not found")).
				AndRunErr("helm --kube-context kubecontext install skaffold-helm-remote stable/chartmuseum --post-renderer SKAFFOLD-BINARY --repo https://charts.helm.sh/stable --kubeconfig kubeconfig", fmt.Errorf("building helm dependencies")),
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
				AndRunEnv("helm --kube-context kubecontext install skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext install skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "get success should upgrade by force, not install",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm --force examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunErr("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/skaffold-helm --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/skaffold-helm --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Sprintf("Packaged to %s", filepath.Join(tmpDir, "skaffold-helm-0.1.2.tgz"))).
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm "+filepath.Join(tmpDir, "skaffold-helm-0.1.2.tgz")+" --post-renderer SKAFFOLD-BINARY --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			shouldErr:          false,
			helm:               testDeployWithPackaged,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "should fail to deploy when packaging fails",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/skaffold-helm --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext package testdata/skaffold-helm --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Errorf("packaging failed")),
			shouldErr:          true,
			helm:               testDeployWithPackaged,
			builds:             testBuilds,
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
				AndRunEnv("helm --kube-context kubecontext upgrade user-skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set image.name=skaffold-helm --set image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set missing.key=<MISSING> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY -f /some/file-FOOBAR.yaml -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm --version 1.0 examples/test --post-renderer SKAFFOLD-BINARY --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			env:                []string{"VERSION=1.0"},
			helm:               testDeployConfigVersionTemplated,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "deploy with templated repo",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --repo https://foo:bar@charts.helm.sh/stable --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			env:                []string{"CHARTUSER=foo", "CHARTPASS=bar"},
			helm:               testDeployConfigRepoTemplated,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "deploy with setFiles",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv(fmt.Sprintf("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set-file expanded=%s --set-file value=/some/file.yaml -f skaffold-overrides.yaml --kubeconfig kubeconfig", strings.ReplaceAll(filepath.Join(home, "file.yaml"), "\\", "\\\\")),
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfigSetFiles,
			builds:             testBuilds,
			expectedNamespaces: []string{""},
		},
		{
			description: "first release without tag, second with tag",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade other examples/test --post-renderer SKAFFOLD-BINARY --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRun("helm --kube-context kubecontext get all other --template {{.Release.Manifest}} --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --debugging --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployConfig,
			builds:             testBuilds,
			configure:          func(deployer *Deployer) { deployer.enableDebug = true },
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
				AndRunEnv("helm --kube-context kubecontext install skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseNamespace --create-namespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
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
				AndRunEnv("helm --kube-context kubecontext upgrade skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --namespace testReleaseNamespace --set some.key=somevalue -f skaffold-overrides.yaml --kubeconfig kubeconfig",
					[]string{"SKAFFOLD_FILENAME=test.yaml", "SKAFFOLD_CMDLINE=filter --kube-context kubecontext --build-artifacts TMPFILE --kubeconfig kubeconfig"}).
				AndRunWithOutput("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --template {{.Release.Manifest}} --kubeconfig kubeconfig", validDeployYaml),
			helm:               testDeployCreateNamespaceConfig,
			builds:             testBuilds,
			expectedNamespaces: []string{"testReleaseNamespace"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&helm.WriteBuildArtifacts, func([]graph.Artifact) (string, func(), error) { return "TMPFILE", func() {}, nil })
			t.Override(&client.Client, deployutil.MockK8sClient)
			fakeWarner := &warnings.Collect{}
			env := test.env
			if env == nil {
				env = []string{"FOO=FOOBAR"}
			}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return env })
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&helm.OSExecutable, func() (string, error) { return "SKAFFOLD-BINARY", nil })
			t.Override(&kubectx.CurrentConfig, func() (api.Config, error) {
				return api.Config{CurrentContext: ""}, nil
			})

			deployer, err := NewDeployer(context.Background(), &helmConfig{
				namespace:  test.namespace,
				force:      test.force,
				configFile: "test.yaml",
			}, &label.DefaultLabeller{}, &test.helm, nil, "default")
			t.RequireNoError(err)

			if test.configure != nil {
				test.configure(deployer)
			}
			deployer.pkgTmpDir = tmpDir
			// Deploy returns nil unless `helm get all <release>` is set up to return actual release info
			err = deployer.Deploy(context.Background(), io.Discard, test.builds, manifest.ManifestListByConfig{})
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedNamespaces, *deployer.namespaces)
		})
	}
}

func TestHelmCleanup(t *testing.T) {
	tests := []struct {
		description      string
		commands         util.Command
		helm             latest.LegacyHelmDeploy
		namespace        string
		builds           []graph.Artifact
		expectedWarnings []string
		dryRun           bool
	}{
		{
			description: "helm3 cleanup dry-run",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get manifest skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
			dryRun: true,
		},
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

			deployer, err := NewDeployer(context.Background(), &helmConfig{
				namespace: test.namespace,
			}, &label.DefaultLabeller{}, &test.helm, nil, "default")
			t.RequireNoError(err)

			deployer.Cleanup(context.Background(), io.Discard, test.dryRun, manifest.ManifestListByConfig{})

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
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			tmpDir := t.NewTempDir().Touch(test.files...)
			var local, remote string
			if test.remote {
				remote = "foo/bar"
			} else {
				local = tmpDir.Root()
			}

			deployer, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &latest.LegacyHelmDeploy{
				Releases: []latest.HelmRelease{{
					Name:                  "skaffold-helm",
					ChartPath:             local,
					RemoteChart:           remote,
					ValuesFiles:           test.valuesFiles,
					Overrides:             schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
					SetValues:             map[string]string{"some.key": "somevalue"},
					SkipBuildDependencies: test.skipBuildDependencies,
				}},
			}, nil, "default")
			t.RequireNoError(err)
			deps, err := deployer.Dependencies()

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected(tmpDir), deps)
		})
	}
}

func TestHelmRender(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		commands    util.Command
		helm        latest.LegacyHelmDeploy
		env         []string
		builds      []graph.Artifact
		namespace   string
	}{
		{
			description: "normal render v3",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated config",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set image.name=skaffold-helm --set image.tag=skaffold-helm:tag1 --set missing.key=<MISSING> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfigTemplated,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated values file",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig"),
			helm: testDeployConfigValuesFilesTemplated,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with missing templated release name should fail",
			shouldErr:   true,
			helm:        testDeployWithTemplatedName,
			builds:      testBuilds,
		},
		{
			description: "render with templated release name",
			env:         []string{"USER=user"},
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template user-skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --kubeconfig kubeconfig"),
			helm:   testDeployWithTemplatedName,
			builds: testBuilds,
		},
		{
			description: "render with templated chart path",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/FOOBAR --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/FOOBAR --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --kubeconfig kubeconfig"),
			helm:   testDeployWithTemplatedChartPath,
			builds: testBuilds,
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			helm: testDeployNamespacedConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --namespace testReleaseFOOBARNamespace --kubeconfig kubeconfig"),
			helm: testDeployEnvTemplateNamespacedConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with remote repo",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployConfigRemoteRepo,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with remote chart",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext template skaffold-helm-remote stable/chartmuseum --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployRemoteChart,
		},
		{
			description: "render with remote chart with version",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext template skaffold-helm-remote stable/chartmuseum --version 1.0.0 --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployRemoteChartVersion,
		},
		{
			description: "render without building chart dependencies",
			shouldErr:   false,
			commands: testutil.
				CmdRun("helm --kube-context kubecontext template skaffold-helm stable/chartmuseum --kubeconfig kubeconfig"),
			helm: testDeploySkipBuildDependencies,
		},
		// https://github.com/GoogleContainerTools/skaffold/issues/7595
		// {
		//	description: "render with cli namespace",
		//	shouldErr:   false,
		//	namespace:   "clinamespace",
		//	commands:    testutil.CmdRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
		//	helm:        testDeployConfig,
		//	builds: []graph.Artifact{
		//		{
		//			ImageName: "skaffold-helm",
		//			Tag:       "skaffold-helm:tag1",
		//		}},
		// },
		// {
		//	description: "render with HelmRelease.Namespace and cli namespace",
		//	shouldErr:   false,
		//	namespace:   "clinamespace",
		//	commands:    testutil.CmdRun("helm --kube-context kubecontext template skaffold-helm examples/test --post-renderer SKAFFOLD-BINARY --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
		//	helm:        testDeployNamespacedConfig,
		//	builds: []graph.Artifact{
		//		{
		//			ImageName: "skaffold-helm",
		//			Tag:       "skaffold-helm:tag1",
		//		}},
		// },
	}
	labeller := label.DefaultLabeller{}
	labels := labeller.Labels()
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return append([]string{"FOO=FOOBAR"}, test.env...) })
			t.Override(&helm.OSExecutable, func() (string, error) { return "SKAFFOLD-BINARY", nil })
			t.Override(&util.DefaultExecCommand, test.commands)
			helmRenderer, err := rhelm.New(&helmConfig{
				namespace: test.namespace,
			}, latest.RenderConfig{
				Generate: latest.Generate{Helm: &latest.Helm{Flags: test.helm.Flags, Releases: test.helm.Releases}},
			}, labels, "default", nil)
			t.RequireNoError(err)
			_, err = helmRenderer.Render(context.Background(), io.Discard, test.builds, true)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHelmHooks(t *testing.T) {
	tests := []struct {
		description string
		runner      hooks.Runner
		shouldErr   bool
	}{
		{
			description: "hooks run successfully",
			runner: hooks.MockRunner{
				PreHooks: func(context.Context, io.Writer) error {
					return nil
				},
				PostHooks: func(context.Context, io.Writer) error {
					return nil
				},
			},
		},
		{
			description: "hooks fails",
			runner: hooks.MockRunner{
				PreHooks: func(context.Context, io.Writer) error {
					return errors.New("failed to execute hooks")
				},
				PostHooks: func(context.Context, io.Writer) error {
					return errors.New("failed to execute hooks")
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			t.Override(&hooks.NewDeployRunner, func(*ctl.CLI, latest.DeployHooks, *[]string, logger.Formatter, hooks.DeployEnvOpts) hooks.Runner {
				return test.runner
			})

			k, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig, nil, "default")
			t.RequireNoError(err)
			err = k.PreDeployHooks(context.Background(), io.Discard)
			t.CheckError(test.shouldErr, err)
			err = k.PostDeployHooks(context.Background(), io.Discard)
			t.CheckError(test.shouldErr, err)
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
func (c *helmConfig) GetNamespace() string                                { return c.namespace }
func (c *helmConfig) ConfigurationFile() string                           { return c.configFile }
func (c *helmConfig) PortForwardResources() []*latest.PortForwardResource { return nil }

func TestHasRunnableHooks(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.LegacyHelmDeploy
		expected    bool
	}{
		{
			description: "no hooks defined",
			cfg:         latest.LegacyHelmDeploy{},
		},
		{
			description: "has pre-deploy hook defined",
			cfg: latest.LegacyHelmDeploy{
				LifecycleHooks: latest.DeployHooks{PreHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
		{
			description: "has post-deploy hook defined",
			cfg: latest.LegacyHelmDeploy{
				LifecycleHooks: latest.DeployHooks{PostHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			k, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &test.cfg, nil, "default")
			t.RequireNoError(err)
			actual := k.HasRunnableHooks()
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
