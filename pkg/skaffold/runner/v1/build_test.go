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

package v1

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTest(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *TestBench
		cfg             []*latest.Artifact
		artifacts       []graph.Artifact
		expectedActions []Actions
		shouldErr       bool
	}{
		{
			description: "test no error",
			testBench:   &TestBench{},
			cfg:         []*latest.Artifact{{ImageName: "img1"}, {ImageName: "img2"}},
			artifacts: []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			},
			expectedActions: []Actions{{
				Tested: []string{"img1:tag1", "img2:tag2"},
			}},
		},
		{
			description:     "no artifacts",
			testBench:       &TestBench{},
			artifacts:       []graph.Artifact(nil),
			expectedActions: []Actions{{}},
		},
		{
			description: "missing tag",
			testBench:   &TestBench{},
			cfg:         []*latest.Artifact{{ImageName: "image1"}},
			artifacts:   []graph.Artifact{{ImageName: "image1"}},
			expectedActions: []Actions{{
				Tested: []string{""},
			}},
		},
		{
			description:     "test error",
			testBench:       &TestBench{testErrors: []error{errors.New("")}},
			expectedActions: []Actions{{}},
			shouldErr:       true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runner := createRunner(t, test.testBench, nil, test.cfg, nil)

			err := runner.Test(context.Background(), ioutil.Discard, test.artifacts)

			t.CheckError(test.shouldErr, err)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestBuildTestDeploy(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *TestBench
		shouldErr       bool
		expectedActions []Actions
	}{
		{
			description: "run no error",
			testBench:   &TestBench{},
			expectedActions: []Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
		{
			description:     "run build error",
			testBench:       &TestBench{buildErrors: []error{errors.New("")}},
			shouldErr:       true,
			expectedActions: []Actions{{}},
		},
		{
			description: "run test error",
			testBench:   &TestBench{testErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "run deploy error",
			testBench:   &TestBench{deployErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)

			ctx := context.Background()
			artifacts := []*latest.Artifact{{
				ImageName: "img",
			}}

			runner := createRunner(t, test.testBench, nil, artifacts, nil)
			bRes, err := runner.Build(ctx, ioutil.Discard, artifacts)
			if err == nil {
				err = runner.Test(ctx, ioutil.Discard, bRes)
				if err == nil {
					err = runner.DeployAndLog(ctx, ioutil.Discard, bRes)
				}
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestBuildDryRun(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &TestBench{}
		artifacts := []*latest.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		}
		runner := createRunner(t, testBench, nil, artifacts, nil)
		runner.runCtx.Opts.DryRun = true

		bRes, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

		t.CheckNoError(err)
		t.CheckDeepEqual([]graph.Artifact{
			{ImageName: "img1", Tag: "img1:latest"},
			{ImageName: "img2", Tag: "img2:latest"}}, bRes)
		// Nothing was built, tested or deployed
		t.CheckDeepEqual([]Actions{{}}, testBench.Actions())
	})
}

func TestBuildPushFlag(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &TestBench{}
		artifacts := []*latest.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		}
		runner := createRunner(t, testBench, nil, artifacts, nil)
		runner.runCtx.Opts.PushImages = config.NewBoolOrUndefined(util.BoolPtr(true))

		_, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

		t.CheckNoError(err)
	})
}

func TestDigestSources(t *testing.T) {
	artifacts := []*latest.Artifact{
		{ImageName: "img1"},
	}

	tests := []struct {
		name         string
		digestSource string
		expected     []graph.Artifact
	}{
		{
			name:         "digest source none",
			digestSource: "none",
			expected:     []graph.Artifact{},
		},
		{
			name:         "digest source tag",
			digestSource: "tag",
			expected: []graph.Artifact{
				{ImageName: "img1", Tag: "img1:latest"},
			},
		},
		{
			name:         "digest source remote",
			digestSource: "remote",
			expected: []graph.Artifact{
				{ImageName: "img1", Tag: "img1:latest"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			testBench := &TestBench{}
			runner := createRunner(t, testBench, nil, artifacts, nil)
			runner.runCtx.Opts.DigestSource = test.digestSource
			runner.runCtx.Opts.RenderOnly = true

			bRes, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, bRes)
			t.CheckDeepEqual([]Actions{{}}, testBench.Actions())
		})
	}
}

func TestCheckWorkspaces(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch("file")
	tmpFile := tmpDir.Path("file")

	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
	}{
		{
			description: "no workspace",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
				},
			},
		},
		{
			description: "directory that exists",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: tmpDir.Root(),
				},
			},
		},
		{
			description: "error on non-existent location",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: "doesnotexist",
				},
			},
			shouldErr: true,
		},
		{
			description: "error on file",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: tmpFile,
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := runner.CheckWorkspaces(test.artifacts)
			t.CheckError(test.shouldErr, err)
		})
	}
}
