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
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
	"github.com/spf13/afero"
)

const testKubeContext = "kubecontext"

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
        image: leeroy-web-image
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
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []config.Manifest{
							{
								Paths:  []string{"test/deployment.yaml"},
								Images: []string{"leeroy-web-image"},
							},
						},
					},
				},
			},
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "leeroy-web-image",
						Tag:       "leeroy-web-image:v1",
					},
				},
			},
		},
		{
			description: "missing manifest file",
			shouldErr:   true,
			cfg: &config.DeployConfig{
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []config.Manifest{
							{
								Paths:  []string{"test/not_deployment.yaml"},
								Images: []string{"leeroy-web-image"},
							},
						},
					},
				},
			},
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "leeroy-web-image",
						Tag:       "leeroy-web-image:123",
					},
				},
			},
		},
		{
			description: "deploy success",
			cfg: &config.DeployConfig{
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []config.Manifest{
							{
								Paths:  []string{"test/deployment.yaml"},
								Images: []string{"leeroy-web-image"},
							},
						},
					},
				},
			},
			command: testutil.NewFakeRunCommand("", "", nil),
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "leeroy-web-image",
						Tag:       "leeroy-web-image:123",
					},
				},
			},
			expected: &Result{},
		},
		{
			description: "deploy command error",
			shouldErr:   true,
			cfg: &config.DeployConfig{
				DeployType: config.DeployType{
					KubectlDeploy: &config.KubectlDeploy{
						Manifests: []config.Manifest{
							{
								Paths:  []string{"test/not_deployment.yaml"},
								Images: []string{"leeroy-web-image"},
							},
						},
					},
				},
			},
			command: testutil.NewFakeRunCommand("", "", fmt.Errorf("")),
			b: &build.BuildResult{
				Builds: []build.Build{
					{
						ImageName: "leeroy-web-image",
						Tag:       "leeroy-web-image:123",
					},
				},
			},
		},
	}

	util.Fs = afero.NewMemMapFs()
	defer util.ResetFs()
	util.Fs.MkdirAll("test", 0750)
	files := map[string]string{
		"test/deployment.yaml": deploymentYAML,
	}
	for path, contents := range files {
		afero.WriteFile(util.Fs, path, []byte(contents), 0644)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				util.DefaultExecCommand = test.command
				defer util.ResetDefaultExecCommand()
			}

			k := NewKubectlDeployer(test.cfg, testKubeContext)
			res, err := k.Deploy(context.Background(), &bytes.Buffer{}, test.b)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
		})

	}
}

func TestReplaceParameters(t *testing.T) {
	manifest := "[IMAGE_NAME][IMAGE_NAME_OTHER][OTHER]"
	expectedManifest := "[image:v1][image_other:v1][other:v1]"

	manifest = replaceParameters(manifest, map[string]build.Build{
		"IMAGE_NAME":       {Tag: "image:v1"},
		"IMAGE_NAME_OTHER": {Tag: "image_other:v1"},
		"OTHER":            {Tag: "other:v1"},
	})

	if manifest != expectedManifest {
		t.Errorf("Expected: '%s'. Got: '%s'", expectedManifest, manifest)
	}
}
