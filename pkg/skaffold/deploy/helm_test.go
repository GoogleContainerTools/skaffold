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
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
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
		Values: map[string]string{
			"image": "skaffold-helm",
		},
		Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
		SetValues: map[string]string{
			"some.key": "somevalue",
		},
	}},
}

var testDeployConfigTemplated = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "skaffold-helm",
		ChartPath: "examples/test",
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
			"image": "skaffold-helm-unmatched",
		}},
	},
}

var testDeployFooWithPackaged = latest.HelmDeploy{
	Releases: []latest.HelmRelease{{
		Name:      "foo",
		ChartPath: "testdata/foo",
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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
		Values: map[string]string{
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

func TestHelmDeploy(t *testing.T) {
	tmpDir := os.TempDir()
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
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy success with recreatePods",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --recreate-pods examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployRecreatePodsConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy success with skipBuildDependencies",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeploySkipBuildDependenciesConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy should error for unmatched parameter",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigParameterUnmatched, false),
			builds:     testBuilds,
			shouldErr:  true,
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm stable/chartmuseum --namespace testNamespace --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeploySkipBuildDependencies, false),
			builds:     testBuilds,
		},
		{
			description: "deploy error remote chart without skipBuildDependencies",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm-remote --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build stable/chartmuseum --kubeconfig kubeconfig", fmt.Errorf("building helm dependencies")),
			runContext: makeRunContext(testDeployRemoteChart, false),
			builds:     testBuilds,
			shouldErr:  true,
		},
		{
			description: "get failure should install not upgrade",
			commands: testutil.
				CmdRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "get failure should install not upgrade with helm image strategy",
			commands: testutil.
				CmdRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image.repository=docker.io:5000/skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployHelmStyleConfig, false),
			builds:     testBuilds,
		},
		{
			description: "helm image strategy with explicit registry should set the Helm registry value",
			commands: testutil.
				CmdRunErr("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig", fmt.Errorf("not found")).
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext install --name skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image.registry=docker.io:5000,image.repository=skaffold-helm,image.tag=3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployHelmExplicitRegistryStyleConfig, false),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade by force, not install",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm --force examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, true),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade without force, not install",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy error",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "dep build error",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig", fmt.Errorf("unexpected error")).
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "should package chart and deploy",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunWithOutput("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", "Packaged to "+filepath.Join(tmpDir, "foo-0.1.2.tgz")).
				AndRun("helm --kube-context kubecontext upgrade foo " + filepath.Join(tmpDir, "foo-0.1.2.tgz") + " --namespace testNamespace --set-string image=foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig"),
			shouldErr:  false,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "should fail to deploy when packaging fails",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get foo --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build testdata/foo --kubeconfig kubeconfig").
				AndRunErr("helm --kube-context kubecontext package testdata/foo --destination "+tmpDir+" --version 0.1.2 --app-version 1.2.3 --kubeconfig kubeconfig", fmt.Errorf("packaging failed")),
			shouldErr:  true,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "deploy and get templated release name",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get <no value>-skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade <no value>-skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get <no value>-skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployWithTemplatedName, false),
			builds:     testBuilds,
		},
		{
			description: "deploy with templated values",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set image.name=skaffold-helm --set image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set missing.key=<no value> --set other.key=FOOBAR --set some.key=somevalue --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigTemplated, false),
			builds:     testBuilds,
		},
		{
			description: "deploy with valuesFiles templated",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace -f skaffold-overrides.yaml --set-string image=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig"),
			runContext: makeRunContext(testDeployConfigValuesFilesTemplated, false),
			builds:     testBuilds,
		},
		{
			description: "deploy without actual tags",
			commands: testutil.
				CmdRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm examples/test --namespace testNamespace --kubeconfig kubeconfig").
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
				CmdRun("helm --kube-context kubecontext get other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build examples/test --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade other examples/test --namespace testNamespace --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get other --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext get skaffold-helm --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext dep build  --kubeconfig kubeconfig").
				AndRun("helm --kube-context kubecontext upgrade skaffold-helm  --namespace testNamespace --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --kubeconfig kubeconfig").
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

			event.InitializeState(test.runContext.Cfg.Build)

			deployer := NewHelmDeployer(test.runContext)
			result := deployer.Deploy(context.Background(), ioutil.Discard, test.builds, nil)

			t.CheckError(test.shouldErr, result.GetError())
			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}

func TestPackageHelmChart(t *testing.T) {

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
			description:           "charts dir is included when skipBuildDependencies is true",
			files:                 []string{"Chart.yaml", "charts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: true,
			expected: func(folder *testutil.TempDir) []string {
				return []string{folder.Path("Chart.yaml"), folder.Path("charts/xyz.tar"), folder.Path("templates/deploy.yaml")}
			},
		},
		{
			description:           "charts dir is excluded when skipBuildDependencies is false",
			files:                 []string{"Chart.yaml", "charts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: false,
			expected: func(folder *testutil.TempDir) []string {
				return []string{folder.Path("Chart.yaml"), folder.Path("templates/deploy.yaml")}
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
					Values:                map[string]string{"image": "skaffold-helm"},
					Overrides:             schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
					SetValues:             map[string]string{"some.key": "somevalue"},
					SkipBuildDependencies: test.skipBuildDependencies,
					Remote:                test.remote,
				},
				},
			}, false))

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
	}{
		{
			description: "calling render returns error",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deployer := NewHelmDeployer(&runcontext.RunContext{})
			actual := deployer.Render(context.Background(), ioutil.Discard, []build.Artifact{}, nil, "tmp/dir")
			t.CheckError(test.shouldErr, actual)
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
			Namespace:  testNamespace,
			KubeConfig: testKubeConfig,
			Force:      force,
		},
	}
}
