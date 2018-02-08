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
	"fmt"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
	"github.com/spf13/afero"
)

const deploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: leeroy-web
  labels:
    app: leeroy-web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-web
  template:
    metadata:
      labels:
        app: leeroy-web
    spec:
      containers:
      - name: leeroy-web
        image: IMAGE_NAME
        ports:
		- containerPort: 8080
`

func TestKubectlRun(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *config.DeployConfig
		b           *build.BuildResult
		command     util.Command

		expected  *Result
		shouldErr bool
	}{
		{
			description: "parameter mismatch",
			shouldErr:   true,
			cfg: &config.DeployConfig{
				Parameters: map[string]string{
					"IMAGE_NAME": "abc",
				},
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []string{
							"test/deployment.yaml",
						},
					},
				},
			},
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "not_abc",
						Tag:       "not_abc:123",
					},
				},
			},
		},
		{
			description: "missing manifest file",
			shouldErr:   true,
			cfg: &config.DeployConfig{
				Parameters: map[string]string{
					"IMAGE_NAME": "abc",
				},
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []string{
							"test/not_deployment.yaml",
						},
					},
				},
			},
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "abc",
						Tag:       "abc:123",
					},
				},
			},
		},
		{
			description: "deploy success",
			cfg: &config.DeployConfig{
				Parameters: map[string]string{
					"IMAGE_NAME": "abc",
				},
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []string{
							"test/deployment.yaml",
						},
					},
				},
			},
			command: testutil.NewFakeRunCommand("", "", nil),
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "abc",
						Tag:       "abc:123",
					},
				},
			},
			expected: &Result{},
		},
		{
			description: "deploy command error",
			shouldErr:   true,
			cfg: &config.DeployConfig{
				Parameters: map[string]string{
					"IMAGE_NAME": "abc",
				},
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []string{
							"test/deployment.yaml",
						},
					},
				},
			},
			command: testutil.NewFakeRunCommand("", "", fmt.Errorf("")),
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "abc",
						Tag:       "abc:123",
					},
				},
			},
		},
	}

	pkgFS := fs
	defer func() {
		fs = pkgFS
	}()
	fs = afero.NewMemMapFs()
	fs.MkdirAll("test", 0750)
	files := map[string]string{
		"test/deployment.yaml": deploymentYAML,
	}
	for path, contents := range files {
		afero.WriteFile(fs, path, []byte(contents), 0644)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				util.DefaultExecCommand = test.command
				defer util.ResetDefaultExecCommand()
			}
			k, err := NewKubectlDeployer(test.cfg)
			if err != nil {
				t.Errorf("Error getting kubectl deployer: %s", err)
				return
			}
			res, err := k.Run(test.b)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
		})

	}
}

func TestDeployManifest(t *testing.T) {
	var tests = []struct {
		description string
		r           io.Reader
		params      map[string]build.Build

		shouldErr bool
	}{
		{
			description: "bad reader",
			r:           testutil.BadReader{},
			params: map[string]build.Build{
				"IMAGE_NAME": {
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:a1b2c3",
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := deployManifest(test.r, test.params)
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
