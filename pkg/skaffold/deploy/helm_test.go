/*
Copyright 2018 The Skaffold Authors

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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/sirupsen/logrus"
)

var testBuilds = []build.Artifact{
	{
		ImageName: "skaffold-helm",
		Tag:       "docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
	},
}

var testBuildsFoo = []build.Artifact{
	{
		ImageName: "foo",
		Tag:       "foo:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
	},
}

var testDeployConfig = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "skaffold-helm",
			ChartPath: "examples/test",
			Values: map[string]string{
				"image": "skaffold-helm",
			},
			Overrides: map[string]interface{}{
				"foo": "bar",
			},
			SetValues: map[string]string{
				"some.key": "somevalue",
			},
		},
	},
}

var testDeployRecreatePodsConfig = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "skaffold-helm",
			ChartPath: "examples/test",
			Values: map[string]string{
				"image": "skaffold-helm",
			},
			Overrides: map[string]interface{}{
				"foo": "bar",
			},
			SetValues: map[string]string{
				"some.key": "somevalue",
			},
			RecreatePods: true,
		},
	},
}

var testDeployHelmStyleConfig = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "skaffold-helm",
			ChartPath: "examples/test",
			Values: map[string]string{
				"image": "skaffold-helm",
			},
			Overrides: map[string]interface{}{
				"foo": "bar",
			},
			SetValues: map[string]string{
				"some.key": "somevalue",
			},
			ImageStrategy: latest.HelmImageStrategy{
				HelmImageConfig: latest.HelmImageConfig{
					HelmConventionConfig: &latest.HelmConventionConfig{},
				},
			},
		},
	},
}

var testDeployConfigParameterUnmatched = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "skaffold-helm",
			ChartPath: "examples/test",
			Values: map[string]string{
				"image": "skaffold-helm-unmatched",
			},
		},
	},
}

var testDeployFooWithPackaged = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "foo",
			ChartPath: "testdata/foo",
			Values: map[string]string{
				"image": "foo",
			},
			Packaged: &latest.HelmPackaged{
				Version:    "0.1.2",
				AppVersion: "1.2.3",
			},
		},
	},
}

var testDeployWithTemplatedName = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "{{.USER}}-skaffold-helm",
			ChartPath: "examples/test",
			Values: map[string]string{
				"image.tag": "skaffold-helm",
			},
			Overrides: map[string]interface{}{
				"foo": "bar",
			},
			SetValues: map[string]string{
				"some.key": "somevalue",
			},
		},
	},
}

var testDeploySkipBuildDependencies = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:                  "skaffold-helm",
			ChartPath:             "stable/chartmuseum",
			SkipBuildDependencies: true,
		},
	},
}

var testDeployRemoteChart = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:                  "skaffold-helm-remote",
			ChartPath:             "stable/chartmuseum",
			SkipBuildDependencies: false,
		},
	},
}

var testNamespace = "testNamespace"

var validDeployYaml = `
# Source: skaffold-helm/templates/deployment.yaml
apiVersion: extensions/v1beta1
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

// TestMain disables logrus output before running tests.
func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestHelmDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cmd         util.Command
		deployer    *HelmDeployer
		builds      []build.Artifact
		shouldErr   bool
	}{
		{
			description: "deploy success",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext, testNamespace, ""),
			builds:      testBuilds,
		},
		{
			description: "deploy success with recreatePods",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployRecreatePodsConfig, testKubeContext, testNamespace, ""),
			builds:      testBuilds,
		},
		{
			description: "deploy error unmatched parameter",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfigParameterUnmatched, testKubeContext, testNamespace, ""),
			builds:      testBuilds,
			shouldErr:   true,
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeploySkipBuildDependencies, testKubeContext, testNamespace, ""),
			builds:      testBuilds,
		},
		{
			description: "deploy error remote chart without skipBuildDependencies",
			cmd: &MockHelm{
				t:         t,
				depResult: fmt.Errorf("unexpected error"),
			},
			deployer:  NewHelmDeployer(testDeployRemoteChart, testKubeContext, testNamespace, ""),
			builds:    testBuilds,
			shouldErr: true,
		},
		{
			description: "get failure should install not upgrade",
			cmd: &MockHelm{
				t:         t,
				getResult: fmt.Errorf("not found"),
				installMatcher: func(cmd *exec.Cmd) bool {
					expected := map[string]bool{fmt.Sprintf("image=%s", testBuilds[0].Tag): true}
					for _, arg := range cmd.Args {
						if expected[arg] {
							return true
						}
					}
					return false
				},
				upgradeResult: fmt.Errorf("should not have called upgrade"),
			},
			deployer: NewHelmDeployer(testDeployConfig, testKubeContext, testNamespace, ""),
			builds:   testBuilds,
		},
		{
			description: "get failure should install not upgrade with helm image strategy",
			cmd: &MockHelm{
				t:         t,
				getResult: fmt.Errorf("not found"),
				installMatcher: func(cmd *exec.Cmd) bool {
					dockerRef, err := docker.ParseReference(testBuilds[0].Tag)
					if err != nil {
						return false
					}

					expected := map[string]bool{fmt.Sprintf("image.repository=%s,image.tag=%s", dockerRef.BaseName, dockerRef.Tag): true}
					for _, arg := range cmd.Args {
						if expected[arg] {
							return true
						}
					}
					return false
				},
				upgradeResult: fmt.Errorf("should not have called upgrade"),
			},
			deployer: NewHelmDeployer(testDeployHelmStyleConfig, testKubeContext, testNamespace, ""),
			builds:   testBuilds,
		},
		{
			description: "get success should upgrade not install",
			cmd: &MockHelm{
				t:             t,
				installResult: fmt.Errorf("should not have called install"),
			},
			deployer: NewHelmDeployer(testDeployConfig, testKubeContext, testNamespace, ""),
			builds:   testBuilds,
		},
		{
			description: "deploy error",
			cmd: &MockHelm{
				t:             t,
				upgradeResult: fmt.Errorf("unexpected error"),
			},
			shouldErr: true,
			deployer:  NewHelmDeployer(testDeployConfig, testKubeContext, testNamespace, ""),
			builds:    testBuilds,
		},
		{
			description: "dep build error",
			cmd: &MockHelm{
				t:         t,
				depResult: fmt.Errorf("unexpected error"),
			},
			shouldErr: true,
			deployer:  NewHelmDeployer(testDeployConfig, testKubeContext, testNamespace, ""),
			builds:    testBuilds,
		},
		{
			description: "should package chart and deploy",
			cmd: &MockHelm{
				t:          t,
				packageOut: bytes.NewBufferString("Packaged to " + os.TempDir() + "foo-0.1.2.tgz"),
			},
			shouldErr: false,
			deployer: NewHelmDeployer(
				testDeployFooWithPackaged,
				testKubeContext,
				testNamespace,
				"",
			),
			builds: testBuildsFoo,
		},
		{
			description: "should fail to deploy when packaging fails",
			cmd: &MockHelm{
				t:             t,
				packageResult: fmt.Errorf("packaging failed"),
			},
			shouldErr: true,
			deployer: NewHelmDeployer(
				testDeployFooWithPackaged,
				testKubeContext,
				testNamespace,
				"",
			),
			builds: testBuildsFoo,
		},
		{
			description: "deploy and get templated release name",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployWithTemplatedName, testKubeContext, testNamespace, ""),
			builds:      testBuilds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = tt.cmd

			err := tt.deployer.Deploy(context.Background(), ioutil.Discard, tt.builds, nil)

			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}

type CommandMatcher func(*exec.Cmd) bool

type MockHelm struct {
	t *testing.T

	getResult      error
	getMatcher     CommandMatcher
	installResult  error
	installMatcher CommandMatcher
	upgradeResult  error
	upgradeMatcher CommandMatcher
	depResult      error

	packageOut    io.Reader
	packageResult error
}

func (m *MockHelm) RunCmdOut(c *exec.Cmd) ([]byte, error) {
	m.t.Error("Shouldn't be used")
	return nil, nil
}

func (m *MockHelm) RunCmd(c *exec.Cmd) error {
	if len(c.Args) < 3 {
		m.t.Errorf("Not enough args in command %v", c)
	}

	if c.Args[1] != "--kube-context" || c.Args[2] != testKubeContext {
		m.t.Errorf("Invalid kubernetes context %v", c)
	}

	if c.Args[3] == "get" || c.Args[3] == "upgrade" {
		if releaseName := c.Args[4]; strings.Contains(releaseName, "{{") {
			m.t.Errorf("Invalid release name: %v", releaseName)
		}
	}

	switch c.Args[3] {
	case "get":
		if m.getMatcher != nil && !m.getMatcher(c) {
			m.t.Errorf("get matcher failed to match cmd")
		}
		return m.getResult
	case "install":
		if m.installMatcher != nil && !m.installMatcher(c) {
			m.t.Errorf("install matcher failed to match cmd")
		}
		return m.installResult
	case "upgrade":
		if m.upgradeMatcher != nil && !m.upgradeMatcher(c) {
			m.t.Errorf("upgrade matcher failed to match cmd")
		}
		return m.upgradeResult
	case "dep":
		return m.depResult
	case "package":
		if m.packageOut != nil {
			if _, err := io.Copy(c.Stdout, m.packageOut); err != nil {
				m.t.Errorf("Failed to copy stdout")
			}
		}
		return m.packageResult
	default:
		m.t.Errorf("Unknown helm command: %+v", c)
		return nil
	}
}

func TestParseHelmRelease(t *testing.T) {
	var tests = []struct {
		name      string
		yaml      []byte
		shouldErr bool
	}{
		{
			name: "parse valid deployment yaml",
			yaml: []byte(validDeployYaml),
		},
		{
			name: "parse valid service yaml",
			yaml: []byte(validServiceYaml),
		},
		{
			name:      "parse invalid deployment yaml",
			yaml:      []byte(invalidDeployYaml),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRuntimeObject(testNamespace, tt.yaml)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}

func TestExtractChartFilename(t *testing.T) {
	out, err := extractChartFilename(
		"Successfully packaged chart and saved it to: /var/folders/gm/rrs_712142x8vymmd7xq7h340000gn/T/foo-1.2.3-dirty.tgz\n",
		"/var/folders/gm/rrs_712142x8vymmd7xq7h340000gn/T/",
	)

	testutil.CheckErrorAndDeepEqual(t, false, err, "foo-1.2.3-dirty.tgz", out)
}

func TestHelmDependencies(t *testing.T) {
	var tests = []struct {
		description string
		files       []string
		valuesFiles []string
		expected    func(folder *testutil.TempDir) []string
	}{
		{
			description: "charts dir is excluded",
			files:       []string{"Chart.yaml", "charts/xyz.tar", "templates/deploy.yaml"},
			expected: func(folder *testutil.TempDir) []string {
				return []string{folder.Path("Chart.yaml"), folder.Path("templates/deploy.yaml")}
			},
		},
		{
			description: "values file is included",
			files:       []string{"Chart.yaml"},
			valuesFiles: []string{"/folder/values.yaml"},
			expected: func(folder *testutil.TempDir) []string {
				return []string{"/folder/values.yaml", folder.Path("Chart.yaml")}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()
			for _, file := range tt.files {
				folder.Write(file, "")
			}

			deployer := NewHelmDeployer(&latest.HelmDeploy{
				Releases: []latest.HelmRelease{
					{
						Name:        "skaffold-helm",
						ChartPath:   folder.Root(),
						ValuesFiles: tt.valuesFiles,
						Values:      map[string]string{"image": "skaffold-helm"},
						Overrides:   map[string]interface{}{"foo": "bar"},
						SetValues:   map[string]string{"some.key": "somevalue"},
					},
				},
			}, testKubeContext, testNamespace, "")

			deps, err := deployer.Dependencies()

			testutil.CheckErrorAndDeepEqual(t, false, err, tt.expected(folder), deps)
		})
	}
}
