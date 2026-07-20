/*
Copyright 2024 The Skaffold Authors

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
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestRenderWithGCBRepositoryRemoteDependency(t *testing.T) {
	t.Skip("Skipping these tests - does not work on new Kokoro instances")
	tests := []struct {
		description    string
		configFile     string
		shouldErr      bool
		expectedOutput string
		expectedErrMsg string
	}{
		{
			description: "GCB repository remote dependency with private git repo",
			configFile: `apiVersion: skaffold/v4beta10
kind: Config
requires:
  - googleCloudBuildRepoV2:
      projectID: k8s-skaffold
      region: us-central1
      connection: github-connection-e2e-tests
      repo: skaffold-getting-started
`,
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: skaffold-example:fixed
    name: getting-started
`,
		},
		{
			description: "GCB repository remote dependency with private git repo, pointing to an specific branch",
			configFile: `apiVersion: skaffold/v4beta10
kind: Config
requires:
  - googleCloudBuildRepoV2:
      projectID: k8s-skaffold
      region: us-central1
      connection: github-connection-e2e-tests
      repo: skaffold-getting-started
      ref: feature-branch
`,
			expectedOutput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  labels:
    app: my-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-deployment
  template:
    metadata:
      labels:
        app: my-deployment
    spec:
      containers:
      - name: getting-started
        image: skaffold-example-deployment:fixed
`,
		},
		{
			description: "GCB repository remote dependency with private git repo fails, bad configuration",
			configFile: `apiVersion: skaffold/v4beta10
kind: Config
requires:
  - googleCloudBuildRepoV2:
      projectID: bad-repo
      region: us-central1
      connection: github-connection-e2e-tests
      repo: skaffold-getting-started
      ref: feature-branch
`,
			shouldErr:      true,
			expectedErrMsg: "getting GCB repo info for skaffold-getting-started: failed to get remote URI for repository skaffold-getting-started",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)
			tmpDir := t.NewTempDir()
			tmpDir.Write("skaffold.yaml", test.configFile)
			args := []string{"--remote-cache-dir", tmpDir.Root(), "--tag", "fixed", "--default-repo=", "--digest-source", "tag"}
			output, err := skaffold.Render(args...).InDir(tmpDir.Root()).RunWithCombinedOutput(t.T)

			t.CheckError(test.shouldErr, err)

			if !test.shouldErr {
				t.CheckDeepEqual(test.expectedOutput, string(output), testutil.YamlObj(t.T))
			} else {
				t.CheckContains(test.expectedErrMsg, string(output))
			}
		})
	}
}

func TestRenderWithRemoteGCS(t *testing.T) {
	tests := []struct {
		description    string
		configFile     string
		args           []string
		shouldErr      bool
		expectedOutput string
		expectedErrMsg string
	}{
		{
			description: "download all repo with same folders from subfolder",
			configFile: `apiVersion: skaffold/v4beta11
kind: Config
requires:
  - googleCloudStorage:
      source:  gs://skaffold-remote-dependency-e2e-tests/test1/*
      path: ./skaffold.yaml
`,
			args: []string{"--tag", "fixed", "--default-repo=", "--digest-source", "tag"},
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - image: skaffold-example:fixed
      name: getting-started`,
		},
		{
			description: "download full repo with top sub folder",
			configFile: `apiVersion: skaffold/v4beta11
kind: Config
requires:
  - googleCloudStorage:
      source:  gs://skaffold-remote-dependency-e2e-tests/test1
      path: ./test1/skaffold.yaml
`,
			args: []string{"--tag", "fixed", "--default-repo=", "--digest-source", "tag"},
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - image: skaffold-example:fixed
      name: getting-started`,
		},
		{
			description: "download full repo with bucket name as top folder",
			configFile: `apiVersion: skaffold/v4beta11
kind: Config
requires:
  - googleCloudStorage:
      source:  gs://skaffold-remote-dependency-e2e-tests
      path: ./skaffold-remote-dependency-e2e-tests/test1/skaffold.yaml
`,
			args: []string{"--tag", "fixed", "--default-repo=", "--digest-source", "tag"},
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - image: skaffold-example:fixed
      name: getting-started`,
		},
		{
			description: "download only all yaml files across bucket",
			configFile: `apiVersion: skaffold/v4beta11
kind: Config
requires:
  - googleCloudStorage:
      source:  gs://skaffold-remote-dependency-e2e-tests/test1/**.yaml
      path: ./skaffold.yaml
`,
			args: []string{"--tag", "fixed", "--default-repo=", "--digest-source", "tag", "-p", "flat-structure"},
			expectedOutput: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
    - image: skaffold-example:fixed
      name: getting-started`,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, NeedsGcp)
			tmpDirRemoteRepo := t.NewTempDir()
			tmpDirTest := t.NewTempDir()

			tmpDirTest.Write("skaffold.yaml", test.configFile)
			args := append(test.args, "--remote-cache-dir", tmpDirRemoteRepo.Root())
			output, err := skaffold.Render(args...).InDir(tmpDirTest.Root()).RunWithCombinedOutput(t.T)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOutput, string(output), testutil.YamlObj(t.T))
		})
	}
}
