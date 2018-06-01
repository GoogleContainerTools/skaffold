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
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testBuilds = []build.Build{
	{
		ImageName: "skaffold-helm",
		Tag:       "skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
	},
}

var testDeployConfig = &v1alpha2.DeployConfig{
	DeployType: v1alpha2.DeployType{
		HelmDeploy: &v1alpha2.HelmDeploy{
			Releases: []v1alpha2.HelmRelease{
				{
					Name:      "skaffold-helm",
					ChartPath: "examples/test",
					Values: map[string]string{
						"image.tag": "skaffold-helm",
					},
					SetValues: map[string]string{
						"some.key": "somevalue",
					},
				},
			},
		},
	},
}

var testDeployConfigParameterUnmatched = &v1alpha2.DeployConfig{
	DeployType: v1alpha2.DeployType{
		HelmDeploy: &v1alpha2.HelmDeploy{
			Releases: []v1alpha2.HelmRelease{
				{
					Name:      "skaffold-helm",
					ChartPath: "examples/test",
					Values: map[string]string{
						"image.tag": "skaffold-helm-unmatched",
					},
				},
			},
		},
	},
}

func TestHelmDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cmd         util.Command
		deployer    *HelmDeployer
		builds      []build.Build
		shouldErr   bool
	}{
		{
			description: "deploy success",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			builds:      testBuilds,
		},
		{
			description: "deploy error unmatched parameter",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfigParameterUnmatched, testKubeContext),
			builds:      testBuilds,
			shouldErr:   true,
		},
		{
			description: "get failure should install not upgrade",
			cmd: &MockHelm{
				t:             t,
				getResult:     fmt.Errorf("not found"),
				upgradeResult: fmt.Errorf("should not have called upgrade"),
			},
			deployer: NewHelmDeployer(testDeployConfig, testKubeContext),
			builds:   testBuilds,
		},
		{
			description: "get success should upgrade not install",
			cmd: &MockHelm{
				t:             t,
				installResult: fmt.Errorf("should not have called install"),
			},
			deployer: NewHelmDeployer(testDeployConfig, testKubeContext),
			builds:   testBuilds,
		},
		{
			description: "deploy error",
			cmd: &MockHelm{
				t:             t,
				upgradeResult: fmt.Errorf("unexpected error"),
			},
			shouldErr: true,
			deployer:  NewHelmDeployer(testDeployConfig, testKubeContext),
			builds:    testBuilds,
		},
		{
			description: "dep build error",
			cmd: &MockHelm{
				t:         t,
				depResult: fmt.Errorf("unexpected error"),
			},
			shouldErr: true,
			deployer:  NewHelmDeployer(testDeployConfig, testKubeContext),
			builds:    testBuilds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = tt.cmd

			err := tt.deployer.Deploy(context.Background(), &bytes.Buffer{}, tt.builds)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}

type MockHelm struct {
	t *testing.T

	getResult     error
	installResult error
	upgradeResult error
	depResult     error
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

	switch c.Args[3] {
	case "get":
		return m.getResult
	case "install":
		return m.installResult
	case "upgrade":
		return m.upgradeResult
	case "dep":
		return m.depResult
	default:
		m.t.Errorf("Unknown helm command: %+v", c)
		return nil
	}
}
