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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
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
			Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
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
			Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
			SetValues: map[string]string{
				"some.key": "somevalue",
			},
			RecreatePods: true,
		},
	},
}

var testDeploySkipBuildDependenciesConfig = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
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
			Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
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
			Overrides: schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
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

var testDeployWithMultipleReleases = &latest.HelmDeploy{
	Releases: []latest.HelmRelease{
		{
			Name:      "release-1",
			ChartPath: "examples/test1",
			Values: map[string]string{
				"image": "skaffold-helm-1",
			},
		}, {
			Name:      "releases-2",
			ChartPath: "examples/test2",
			Values: map[string]string{
				"image": "skaffold-helm-2",
			},
		},
	},
}

var testNamespace = "testNamespace"

// TestMain disables logrus output before running tests.
func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestHelmDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cmd         util.Command
		runContext  *runcontext.RunContext
		builds      []build.Artifact
		shouldErr   bool
	}{
		{
			description: "deploy success",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployConfig, false),
			builds:      testBuilds,
		},
		{
			description: "deploy success with recreatePods",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployRecreatePodsConfig, false),
			builds:      testBuilds,
		},
		{
			description: "deploy success with skipBuildDependencies",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeploySkipBuildDependenciesConfig, false),
			builds:      testBuilds,
		},
		{
			description: "deploy error unmatched parameter",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployConfigParameterUnmatched, false),
			builds:      testBuilds,
			shouldErr:   true,
		},
		{
			description: "deploy success remote chart with skipBuildDependencies",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeploySkipBuildDependencies, false),
			builds:      testBuilds,
		},
		{
			description: "deploy error remote chart without skipBuildDependencies",
			cmd: &MockHelm{
				t:         t,
				depResult: fmt.Errorf("unexpected error"),
			},
			runContext: makeRunContext(testDeployRemoteChart, false),
			builds:     testBuilds,
			shouldErr:  true,
		},
		{
			description: "get failure should install not upgrade",
			cmd: &MockHelm{
				t:         t,
				getResult: fmt.Errorf("not found"),
				installMatcher: func(cmd *exec.Cmd) bool {
					expected := fmt.Sprintf("image=%s", testBuilds[0].Tag)
					for _, arg := range cmd.Args {
						if expected == arg {
							return true
						}
					}
					return false
				},
				upgradeResult: fmt.Errorf("should not have called upgrade"),
			},
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
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

					expected := fmt.Sprintf("image.repository=%s,image.tag=%s", dockerRef.BaseName, dockerRef.Tag)
					for _, arg := range cmd.Args {
						if expected == arg {
							return true
						}
					}
					return false
				},
				upgradeResult: fmt.Errorf("should not have called upgrade"),
			},
			runContext: makeRunContext(testDeployHelmStyleConfig, false),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade by force, not install",
			cmd: &MockHelm{
				t: t,
				upgradeMatcher: func(cmd *exec.Cmd) bool {
					for _, arg := range cmd.Args {
						if arg == "--force" {
							return true
						}
					}
					return false
				},
				installResult: fmt.Errorf("should not have called install"),
			},
			runContext: makeRunContext(testDeployConfig, true),
			builds:     testBuilds,
		},
		{
			description: "get success should upgrade without force, not install",
			cmd: &MockHelm{
				t: t,
				upgradeMatcher: func(cmd *exec.Cmd) bool {
					for _, arg := range cmd.Args {
						if arg == "--force" {
							return false
						}
					}
					return true
				},
				installResult: fmt.Errorf("should not have called install"),
			},
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "deploy error",
			cmd: &MockHelm{
				t:             t,
				upgradeResult: fmt.Errorf("unexpected error"),
			},
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "dep build error",
			cmd: &MockHelm{
				t:         t,
				depResult: fmt.Errorf("unexpected error"),
			},
			shouldErr:  true,
			runContext: makeRunContext(testDeployConfig, false),
			builds:     testBuilds,
		},
		{
			description: "should package chart and deploy",
			cmd: &MockHelm{
				t:          t,
				packageOut: bytes.NewBufferString("Packaged to " + os.TempDir() + "foo-0.1.2.tgz"),
			},
			shouldErr:  false,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "should fail to deploy when packaging fails",
			cmd: &MockHelm{
				t:             t,
				packageResult: fmt.Errorf("packaging failed"),
			},
			shouldErr:  true,
			runContext: makeRunContext(testDeployFooWithPackaged, false),
			builds:     testBuildsFoo,
		},
		{
			description: "deploy and get templated release name",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployWithTemplatedName, false),
			builds:      testBuilds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			event.InitializeState(tt.runContext)
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = tt.cmd

			err := NewHelmDeployer(tt.runContext).Deploy(context.Background(), ioutil.Discard, tt.builds, nil)

			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}

type CommandMatcher func(*exec.Cmd) bool

type MockHelm struct {
	t *testing.T

	getResult      error
	getResultBytes func(string) (io.Reader, error)
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

	if c.Args[3] == "upgrade" {
		if releaseName := c.Args[4]; strings.Contains(releaseName, "{{") {
			m.t.Errorf("Invalid release name: %v", releaseName)
		}
	}
	if c.Args[3] == "get" {
		releaseName := c.Args[4]
		if releaseName == "manifest" {
			releaseName = c.Args[5]
		}
		if strings.Contains(releaseName, "{{") {
			m.t.Errorf("Invalid release name: %v", releaseName)
		}
	}

	switch c.Args[3] {
	case "get":
		if m.getMatcher != nil && !m.getMatcher(c) {
			m.t.Errorf("get matcher failed to match cmd")
		}
		if m.getResultBytes != nil {
			b, err := m.getResultBytes(c.Args[5])
			if err != nil {
				return err
			}
			if _, err := io.Copy(c.Stdout, b); err != nil {
				m.t.Errorf("Failed to copy stdout")
			}
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

func TestExtractChartFilename(t *testing.T) {
	out, err := extractChartFilename(
		"Successfully packaged chart and saved it to: /var/folders/gm/rrs_712142x8vymmd7xq7h340000gn/T/foo-1.2.3-dirty.tgz\n",
		"/var/folders/gm/rrs_712142x8vymmd7xq7h340000gn/T/",
	)

	testutil.CheckErrorAndDeepEqual(t, false, err, "foo-1.2.3-dirty.tgz", out)
}

func TestHelmDependencies(t *testing.T) {
	var tests = []struct {
		description           string
		files                 []string
		valuesFiles           []string
		skipBuildDependencies bool
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
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()
			for _, file := range tt.files {
				folder.Write(file, "")
			}
			deployer := NewHelmDeployer(makeRunContext(&latest.HelmDeploy{
				Releases: []latest.HelmRelease{
					{
						Name:                  "skaffold-helm",
						ChartPath:             folder.Root(),
						ValuesFiles:           tt.valuesFiles,
						Values:                map[string]string{"image": "skaffold-helm"},
						Overrides:             schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
						SetValues:             map[string]string{"some.key": "somevalue"},
						SkipBuildDependencies: tt.skipBuildDependencies,
					},
				},
			}, false))

			deps, err := deployer.Dependencies()

			testutil.CheckErrorAndDeepEqual(t, false, err, tt.expected(folder), deps)
		})
	}
}

func makeRunContext(helmDeploy *latest.HelmDeploy, force bool) *runcontext.RunContext {
	return &runcontext.RunContext{
		Cfg: &latest.Pipeline{
			Deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy: helmDeploy,
				},
			},
		},
		KubeContext: testKubeContext,
		Opts: &config.SkaffoldOptions{
			Namespace: testNamespace,
			Force:     force,
		},
	}
}

func TestHelmGetManifestsFromReleases(t *testing.T) {
	var tests = []struct {
		description string
		cmd         util.Command
		runContext  *runcontext.RunContext
		releases    []string
		expected    kubectl.ManifestList
	}{
		{
			description: "get manifests for multiple releases",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployWithMultipleReleases, false),
			releases:    []string{"release-1", "release-2"},
			expected: kubectl.ManifestList{
				[]byte(serviceYaml),
				[]byte(deploymentYaml),
				[]byte(podYaml),
			},
		},
		{
			description: "get manifests for single release",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployConfig, false),
			releases:    []string{"release-1"},
			expected: kubectl.ManifestList{
				[]byte(serviceYaml),
				[]byte(deploymentYaml),
			},
		},
		{
			description: "get manifests for non existing release",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployWithMultipleReleases, false),
			releases:    []string{"does-not-exist"},
			expected:    kubectl.ManifestList{},
		},
		{
			description: "get manifests for no releases",
			cmd:         &MockHelm{t: t},
			runContext:  makeRunContext(testDeployWithMultipleReleases, false),
			releases:    []string{""},
			expected:    kubectl.ManifestList{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			event.InitializeState(tt.runContext)
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = tt.cmd
			tt.cmd.(*MockHelm).getResultBytes = mockReleseInfo
			h := NewHelmDeployer(tt.runContext)
			actual := h.getManifestsFromReleases(context.Background(), tt.releases)
			testutil.CheckDeepEqual(t, tt.expected.String(), actual.String())
		})
	}

}

var (
	releaseMap = map[string]string{
		"release-1": fmt.Sprintf(`---
# Source: skaffold-helm/templates/service.yaml
%s---
# Source: skaffold-helm/templates/deployment.yaml
%s---`, serviceYaml, deploymentYaml),
		"release-2": fmt.Sprintf(`---
		# Source: skaffold-helm/templates/pod.yaml
		%s---`, podYaml),
	}
)

func mockReleseInfo(r string) (io.Reader, error) {
	b, ok := releaseMap[r]
	if !ok {
		return nil, fmt.Errorf("not found release %s", r)
	}
	return strings.NewReader(b), nil
}
