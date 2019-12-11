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

package integration

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectlRender(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	tests := []struct {
		description string
		builds      []build.Artifact
		input       string
		expectedOut string
	}{
		{
			description: "normal render",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/k8s-skaffold/skaffold",
					Tag:       "gcr.io/k8s-skaffold/skaffold:test",
				},
			},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
    name: skaffold
`,
		},
		{
			description: "two artifacts",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "two artifacts, combined manifests",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
---
apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Write("deployment.yaml", test.input).
				Chdir()

			deployer := deploy.NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: []string{"deployment.yaml"},
							},
						},
					},
				},
			})
			var b bytes.Buffer
			err := deployer.Render(context.Background(), &b, test.builds, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, b.String())
		})
	}
}
