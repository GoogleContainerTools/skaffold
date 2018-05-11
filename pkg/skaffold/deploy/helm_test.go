/*
Copyright 2018 Google LLC

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

var testBuildResult = &build.BuildResult{
	Builds: []build.Build{
		{
			ImageName: "skaffold-helm",
			Tag:       "skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
		},
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
		buildResult *build.BuildResult

		shouldErr bool
	}{
		{
			description: "deploy success",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			buildResult: testBuildResult,
		},
		{
			description: "deploy error unmatched parameter",
			cmd:         &MockHelm{t: t},
			deployer:    NewHelmDeployer(testDeployConfigParameterUnmatched, testKubeContext),
			buildResult: testBuildResult,
			shouldErr:   true,
		},
		{
			description: "get failure should install not upgrade",
			cmd: &MockHelm{
				t:             t,
				getResult:     cmdOutput{"", fmt.Errorf("not found")},
				upgradeResult: cmdOutput{"", fmt.Errorf("should not have called upgrade")},
			},
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			buildResult: testBuildResult,
		},
		{
			description: "get success should upgrade not install",
			cmd: &MockHelm{
				t:             t,
				installResult: cmdOutput{"", fmt.Errorf("should not have called install")},
			},
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			buildResult: testBuildResult,
		},
		{
			description: "deploy error",
			cmd: &MockHelm{
				t:             t,
				upgradeResult: cmdOutput{"", fmt.Errorf("unexpected error")},
			},
			shouldErr:   true,
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			buildResult: testBuildResult,
		},
		{
			description: "dep build error",
			cmd: &MockHelm{
				t:         t,
				depResult: cmdOutput{"", fmt.Errorf("unexpected error")},
			},
			shouldErr:   true,
			deployer:    NewHelmDeployer(testDeployConfig, testKubeContext),
			buildResult: testBuildResult,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = tt.cmd

			_, err := tt.deployer.Deploy(context.Background(), &bytes.Buffer{}, tt.buildResult)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}

}

type MockHelm struct {
	getResult     cmdOutput
	installResult cmdOutput
	upgradeResult cmdOutput
	depResult     cmdOutput

	t *testing.T
}

type cmdOutput struct {
	stdout string
	err    error
}

func (c cmdOutput) out() ([]byte, error) {
	return []byte(c.stdout), c.err
}

func (m *MockHelm) RunCmdOut(c *exec.Cmd) ([]byte, error) {
	if len(c.Args) < 3 {
		m.t.Errorf("Not enough args in command %v", c)
	}

	if c.Args[1] != "--kube-context" || c.Args[2] != testKubeContext {
		m.t.Errorf("Invalid kubernetes context %v", c)
	}

	switch c.Args[3] {
	case "get":
		return m.getResult.out()
	case "install":
		return m.installResult.out()
	case "upgrade":
		return m.upgradeResult.out()
	case "dep":
		return m.depResult.out()
	}

	m.t.Errorf("Unknown helm command: %+v", c)
	return nil, nil
}

func (m *MockHelm) RunCmd(c *exec.Cmd) error {
	_, err := m.RunCmdOut(c)
	return err
}
