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
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
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

var testNamespace = "testNamespace"

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
)

func TestHelmDeploy(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "TestHelmDeploy")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}

	tests := []struct {
		description      string
		commands         util.Command
		runContext       *runcontext.RunContext
		builds           []build.Artifact
		shouldErr        bool
		expectedWarnings []string
	}{
		{
			description: "deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.0beta deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30b).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.0beta namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30b).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3.0 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.0 namespaced deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployNamespacedConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.0 namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3.0 namespaced context deploy success overrides release namespaces",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version30).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployNamespacedConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3.1 deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.1 namespaced deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testReleaseNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testReleaseNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployNamespacedConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3.1 namespaced context deploy success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3.1 namespaced context deploy success overrides release namespaces",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all --namespace testNamespace skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployNamespacedConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3 unparseable version",
			commands:    testutil.CmdRunWithOutput("helm version --client", "gobbledygook"),
			runContext:  makeRunContext(testDeployConfig, false),
			builds:      testBuilds,
			shouldErr:   true,
		},
		{
			description: "deploy success with recreatePods",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --recreate-pods examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployRecreatePodsConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy success with skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeploySkipBuildDependenciesConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy should error for unmatched parameter",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigParameterUnmatched, false),
			builds:     testBuilds,
			shouldErr:  true,
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm stable/chartmuseum --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeploySkipBuildDependencies, false),
			builds:     testBuilds,
		},
		{
			description: "deploy success when `upgradeOnChange: false` and does not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm-upgradeOnChange --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployUpgradeOnChange, false),
		},
		{
			description: "deploy error remote chart without skipBuildDependencies",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm-remote --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build stable/chartmuseum --kubeconfig kubeconfig", fmt.Errorf("building helm dependencies")),
			runContext: makeRunContext(testDeployRemoteChart, false),
			builds:     testBuilds,
			shouldErr:  true,
		},
		{
			description: "get failure should install not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3 get failure should install not upgrade",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunErr("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get all skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "get failure should install not upgrade with helm image strategy",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.repository=docker.io:5000/skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployHelmStyleConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm image strategy with explicit registry should set the Helm registry value",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.registry=docker.io:5000,image.repository=skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployHelmExplicitRegistryStyleConfig, false),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade by force, not install",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --force examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, true),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade without force, not install",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy error",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "dep build error",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm 2.0 should package chart and deploy",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Sprintf("Packaged to %s", filepath.Join(tmpDir, "foo-0.1.2.tgz"))).
				AndRun("helm --kube-context kubecontext upgrade foo " + filepath.Join(tmpDir, "foo-0.1.2.tgz") + " --set-string image=foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig"),
			shouldErr:  false,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
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
			shouldErr:  false,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
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
			shouldErr:  false,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "should fail to deploy when packaging fails",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Errorf("packaging failed")),
			shouldErr:  true,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "deploy and get templated release name",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get <no value>-skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade <no value>-skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get <no value>-skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployWithTemplatedName, false),
			builds:     testBuilds,
		},
		{
			description: "deploy with templated values",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set image.name=skaffold-helm --set image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set missing.key=<no value> --set other.key=FOOBAR --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigTemplated, false),
			builds:     testBuilds,
		},
		{
			description: "deploy with valuesFiles templated",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigValuesFilesTemplated, false),
			builds:     testBuilds,
		},
		{
			description: "deploy without actual tags",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployWithoutTags, false),
			builds:     testBuilds,
			expectedWarnings: []string{
				"See helm sample for how to replace image names with their actual tags: https://github.com/GoogleContainerTools/skaffold/blob/master/examples/helm-deployment/skaffold.yaml",
				"image [docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184] is not used.",
				"image [skaffold-helm] is used instead.",
			},
		},
		{
			description: "first release without tag, second with tag",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext get other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade other examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build  --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm  --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testTwoReleases, false),
			builds:     testBuilds,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=FOOBAR"} })
			t.Override(&util.DefaultExecCommand, test.commands)

			deployer := NewHelmDeployer(test.runContext, nil)
			deployer.pkgTmpDir = tmpDir
			_, err := deployer.Deploy(context.Background(), ioutil.Discard, test.builds)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}

func TestHelmCleanup(t *testing.T) {
	tests := []struct {
		description      string
		commands         util.Command
		runContext       *runcontext.RunContext
		builds           []build.Artifact
		shouldErr        bool
		expectedWarnings []string
	}{
		{
			description: "cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version20rc).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --purge --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3 cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3 namespace cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployNamespacedConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployConfig),
			builds:     testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success overriding release namespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			runContext: makeNamespacedRunContext(testDeployNamespacedConfig),
			builds:     testBuilds,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=FOOBAR"} })
			t.Override(&util.DefaultExecCommand, test.commands)

			deployer := NewHelmDeployer(test.runContext, nil)
			err := deployer.Cleanup(context.Background(), ioutil.Discard)
			t.CheckError(test.shouldErr, err)
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
			_, err := parseRuntimeObject(testNamespace, test.yaml)

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
			files:                 []string{"Chart.yaml", "Chart.lock", "requirements.yaml", "requirements.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: true,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.lock"),
					folder.Path("Chart.yaml"),
					folder.Path("charts/xyz.tar"),
					folder.Path("requirements.lock"),
					folder.Path("requirements.yaml"),
					folder.Path("templates/deploy.yaml"),
					folder.Path("tmpcharts/xyz.tar"),
				}
			},
		},
		{
			description:           "charts download dir and lock files are excluded when skipBuildDependencies is false",
			files:                 []string{"Chart.yaml", "Chart.lock", "requirements.yaml", "requirements.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: false,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.yaml"),
					folder.Path("requirements.yaml"),
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
			tmpDir := t.NewTempDir().
				Touch(test.files...)

			deployer := NewHelmDeployer(makeRunContext(latest.HelmDeploy{
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
			}, false), nil)

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
		runContext  *runcontext.RunContext
		outputFile  string
		expected    string
		builds      []build.Artifact
	}{
		{
			description: "error if version can't be retrieved",
			shouldErr:   true,
			commands:    testutil.CmdRunErr("helm version --client", fmt.Errorf("yep not working")),
			runContext:  makeRunContext(testDeployConfig, false),
		},
		{
			description: "normal render v2",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version21).
				AndRun("helm --kube-context kubecontext template examples/test --name skaffold-helm --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds: []build.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "normal render v3",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
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
			runContext: makeRunContext(testDeployConfig, false),
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
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set image.name=skaffold-helm --set image.tag=skaffold-helm:tag1 --set missing.key=<no value> --set other.key=<no value> --set some.key=somevalue --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigTemplated, false),
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
			runContext: makeRunContext(testDeployNamespacedConfig, false),
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

			deployer := NewHelmDeployer(test.runContext, nil)

			t.Override(&util.DefaultExecCommand, test.commands)
			err := deployer.Render(context.Background(), ioutil.Discard, test.builds, true, file)
			t.CheckError(test.shouldErr, err)

			if file != "" {
				dat, _ := ioutil.ReadFile(file)
				t.CheckDeepEqual(string(dat), test.expected)
			}
		})
	}
}

func makeRunContext(deploy latest.HelmDeploy, force bool) *runcontext.RunContext {
	pipeline := latest.Pipeline{}
	pipeline.Deploy.DeployType.HelmDeploy = &deploy

	return &runcontext.RunContext{
		Cfg:         pipeline,
		KubeContext: testKubeContext,
		Opts: config.SkaffoldOptions{
			KubeConfig: testKubeConfig,
			Force:      force,
		},
	}
}

func makeNamespacedRunContext(deploy latest.HelmDeploy) *runcontext.RunContext {
	runContext := makeRunContext(deploy, false)
	runContext.Opts.Namespace = testNamespace

	return runContext
}
