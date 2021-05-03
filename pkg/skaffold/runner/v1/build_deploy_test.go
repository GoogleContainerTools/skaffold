/*
Copyright 2021 The Skaffold Authors

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/test"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTest(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *test.TestBench
		cfg             []*latest_v1.Artifact
		artifacts       []graph.Artifact
		expectedActions []test.Actions
		shouldErr       bool
	}{
		{
			description: "test no error",
			testBench:   &test.TestBench{},
			cfg:         []*latest_v1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}},
			artifacts: []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			},
			expectedActions: []test.Actions{{
				Tested: []string{"img1:tag1", "img2:tag2"},
			}},
		},
		{
			description:     "no artifacts",
			testBench:       &test.TestBench{},
			artifacts:       []graph.Artifact(nil),
			expectedActions: []test.Actions{{}},
		},
		{
			description: "missing tag",
			testBench:   &test.TestBench{},
			cfg:         []*latest_v1.Artifact{{ImageName: "image1"}},
			artifacts:   []graph.Artifact{{ImageName: "image1"}},
			expectedActions: []test.Actions{{
				Tested: []string{""},
			}},
		},
		{
			description:     "test error",
			testBench:       &test.TestBench{TestErrors: []error{errors.New("")}},
			expectedActions: []test.Actions{{}},
			shouldErr:       true,
		},
	}
	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			runner := MockRunnerV1(t, testdata.testBench, nil, testdata.cfg, nil)

			err := runner.Test(context.Background(), ioutil.Discard, testdata.artifacts)

			t.CheckError(testdata.shouldErr, err)

			t.CheckErrorAndDeepEqual(testdata.shouldErr, err, testdata.expectedActions, testdata.testBench.Actions())
		})
	}
}

func TestBuildTestDeploy(t *testing.T) {
	tests := []struct {
		description     string
		testBench       *test.TestBench
		shouldErr       bool
		expectedActions []test.Actions
	}{
		{
			description: "run no error",
			testBench:   &test.TestBench{},
			expectedActions: []test.Actions{{
				Built:    []string{"img:1"},
				Tested:   []string{"img:1"},
				Deployed: []string{"img:1"},
			}},
		},
		{
			description:     "run build error",
			testBench:       &test.TestBench{BuildErrors: []error{errors.New("")}},
			shouldErr:       true,
			expectedActions: []test.Actions{{}},
		},
		{
			description: "run test error",
			testBench:   &test.TestBench{TestErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []test.Actions{{
				Built: []string{"img:1"},
			}},
		},
		{
			description: "run deploy error",
			testBench:   &test.TestBench{DeployErrors: []error{errors.New("")}},
			shouldErr:   true,
			expectedActions: []test.Actions{{
				Built:  []string{"img:1"},
				Tested: []string{"img:1"},
			}},
		},
	}
	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)

			ctx := context.Background()
			artifacts := []*latest_v1.Artifact{{
				ImageName: "img",
			}}

			runner := MockRunnerV1(t, testdata.testBench, nil, artifacts, nil)
			bRes, err := runner.Build(ctx, ioutil.Discard, artifacts)
			if err == nil {
				err = runner.Test(ctx, ioutil.Discard, bRes)
				if err == nil {
					err = runner.DeployAndLog(ctx, ioutil.Discard, bRes)
				}
			}

			t.CheckErrorAndDeepEqual(testdata.shouldErr, err, testdata.expectedActions, testdata.testBench.Actions())
		})
	}
}

func TestBuildDryRun(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &test.TestBench{}
		artifacts := []*latest_v1.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		}
		runner := MockRunnerV1(t, testBench, nil, artifacts, nil)
		runner.RunCtx.Opts.DryRun = true

		bRes, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

		t.CheckNoError(err)
		t.CheckDeepEqual([]graph.Artifact{
			{ImageName: "img1", Tag: "img1:latest"},
			{ImageName: "img2", Tag: "img2:latest"}}, bRes)
		// Nothing was built, tested or deployed
		t.CheckDeepEqual([]test.Actions{{}}, testBench.Actions())
	})
}

func TestBuildPushFlag(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &test.TestBench{}
		artifacts := []*latest_v1.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		}
		runner := MockRunnerV1(t, testBench, nil, artifacts, nil)
		runner.RunCtx.Opts.PushImages = config.NewBoolOrUndefined(util.BoolPtr(true))

		_, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

		t.CheckNoError(err)
	})
}

func TestDigestSources(t *testing.T) {
	artifacts := []*latest_v1.Artifact{
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
	for _, testdata := range tests {
		testutil.Run(t, testdata.name, func(t *testutil.T) {
			testBench := &test.TestBench{}
			runner := MockRunnerV1(t, testBench, nil, artifacts, nil)
			runner.RunCtx.Opts.DigestSource = testdata.digestSource

			bRes, err := runner.Build(context.Background(), ioutil.Discard, artifacts)

			t.CheckNoError(err)
			t.CheckDeepEqual(testdata.expected, bRes)
			t.CheckDeepEqual([]test.Actions{{}}, testBench.Actions())
		})
	}
}

func TestCheckWorkspaces(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch("file")
	tmpFile := tmpDir.Path("file")

	tests := []struct {
		description string
		artifacts   []*latest_v1.Artifact
		shouldErr   bool
	}{
		{
			description: "no workspace",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image",
				},
			},
		},
		{
			description: "directory that exists",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image",
					Workspace: tmpDir.Root(),
				},
			},
		},
		{
			description: "error on non-existent location",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image",
					Workspace: "doesnotexist",
				},
			},
			shouldErr: true,
		},
		{
			description: "error on file",
			artifacts: []*latest_v1.Artifact{
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
