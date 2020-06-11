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

package runner

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

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
			t.Override(&kubernetes.Client, mockK8sClient)

			ctx := context.Background()
			artifacts := []*latest.Artifact{{
				ImageName: "img",
			}}

			runner := createRunner(t, test.testBench, nil)
			bRes, err := runner.BuildAndTest(ctx, ioutil.Discard, artifacts)
			if err == nil {
				err = runner.DeployAndLog(ctx, ioutil.Discard, bRes)
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedActions, test.testBench.Actions())
		})
	}
}

func TestBuildAndTestDryRun(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &TestBench{}
		runner := createRunner(t, testBench, nil)
		runner.runCtx.Opts.DryRun = true

		bRes, err := runner.BuildAndTest(context.Background(), ioutil.Discard, []*latest.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual([]build.Artifact{
			{ImageName: "img1", Tag: "img1:latest"},
			{ImageName: "img2", Tag: "img2:latest"}}, bRes)
		// Nothing was built, tested or deployed
		t.CheckDeepEqual([]Actions{{}}, testBench.Actions())
	})
}

func TestBuildAndTestSkipBuild(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testBench := &TestBench{}
		runner := createRunner(t, testBench, nil)
		runner.runCtx.Opts.DigestSource = "none"

		bRes, err := runner.BuildAndTest(context.Background(), ioutil.Discard, []*latest.Artifact{
			{ImageName: "img1"},
			{ImageName: "img2"},
		})

		t.CheckNoError(err)
		t.CheckDeepEqual([]build.Artifact{}, bRes)
		// Nothing was built, tested or deployed
		t.CheckDeepEqual([]Actions{{}}, testBench.Actions())
	})
}
