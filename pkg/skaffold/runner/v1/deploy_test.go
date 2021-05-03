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
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/test"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeploy(t *testing.T) {
	expectedOutput := "Waiting for deployments to stabilize..."
	tests := []struct {
		description       string
		testBench         *test.TestBench
		statusCheckFlag   *bool // --status-check CLI flag
		statusCheckConfig *bool // skaffold.yaml Deploy.StatusCheck field
		shouldErr         bool
		shouldWait        bool
	}{
		{
			description: "deploy shd perform status check when statusCheck flag is unspecified, in-config value is unspecified",
			testBench:   &test.TestBench{},
			shouldWait:  true,
		},
		{
			description:       "deploy shd not perform status check when statusCheck flag is unspecified, in-config value is false",
			testBench:         &test.TestBench{},
			statusCheckConfig: util.BoolPtr(false),
		},
		{
			description:       "deploy shd perform status check when statusCheck flag is unspecified, in-config value is true",
			testBench:         &test.TestBench{},
			statusCheckConfig: util.BoolPtr(true),
			shouldWait:        true,
		},
		{
			description:     "deploy shd not perform status check when statusCheck flag is false, in-config value is unspecified",
			testBench:       &test.TestBench{},
			statusCheckFlag: util.BoolPtr(false),
		},
		{
			description:       "deploy shd not perform status check when statusCheck flag is false, in-config value is false",
			testBench:         &test.TestBench{},
			statusCheckFlag:   util.BoolPtr(false),
			statusCheckConfig: util.BoolPtr(false),
		},
		{
			description:       "deploy shd not perform status check when statusCheck flag is false, in-config value is true",
			testBench:         &test.TestBench{},
			statusCheckFlag:   util.BoolPtr(false),
			statusCheckConfig: util.BoolPtr(true),
		},
		{
			description:     "deploy shd perform status check when statusCheck flag is true, in-config value is unspecified",
			testBench:       &test.TestBench{},
			statusCheckFlag: util.BoolPtr(true),
			shouldWait:      true,
		},
		{
			description:       "deploy shd perform status check when statusCheck flag is true, in-config value is false",
			testBench:         &test.TestBench{},
			statusCheckFlag:   util.BoolPtr(true),
			statusCheckConfig: util.BoolPtr(false),
			shouldWait:        true,
		},
		{
			description:       "deploy shd perform status check when statusCheck flag is true, in-config value is true",
			testBench:         &test.TestBench{},
			statusCheckFlag:   util.BoolPtr(true),
			statusCheckConfig: util.BoolPtr(true),
			shouldWait:        true,
		},
		{
			description:     "deploy shd not perform status check when deployer is in error",
			testBench:       &test.TestBench{DeployErrors: []error{errors.New("deploy error")}},
			shouldErr:       true,
			statusCheckFlag: util.BoolPtr(true),
		},
	}

	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			t.Override(&newStatusCheck, func(status.Config, *label.DefaultLabeller) status.Checker {
				return dummyStatusChecker{}
			})

			runner := MockRunnerV1(t, testdata.testBench, nil, []*latest_v1.Artifact{{ImageName: "img1"},
				{ImageName: "img2"}}, nil)
			runner.RunCtx.Opts.StatusCheck = config.NewBoolOrUndefined(testdata.statusCheckFlag)
			runner.RunCtx.Pipelines.All()[0].Deploy.StatusCheck = testdata.statusCheckConfig
			out := new(bytes.Buffer)

			err := runner.Deploy(context.Background(), out, []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
			t.CheckError(testdata.shouldErr, err)
			if strings.Contains(out.String(), expectedOutput) != testdata.shouldWait {
				t.Errorf("expected %s to contain %s %t. But found %t", out.String(),
					expectedOutput, testdata.shouldWait, !testdata.shouldWait)
			}
		})
	}
}

func TestDeployNamespace(t *testing.T) {
	tests := []struct {
		description string
		Namespaces  []string
		testBench   *test.TestBench
		expected    []string
	}{
		{
			description: "deploy shd add all namespaces to run Context",
			Namespaces:  []string{"test", "test-ns"},
			testBench:   test.NewTestBench().WithDeployNamespaces([]string{"test-ns", "test-ns-1"}),
			expected:    []string{"test", "test-ns", "test-ns-1"},
		},
		{
			description: "deploy without command opts namespace",
			testBench:   test.NewTestBench().WithDeployNamespaces([]string{"test-ns", "test-ns-1"}),
			expected:    []string{"test-ns", "test-ns-1"},
		},
		{
			description: "deploy with no namespaces returned",
			Namespaces:  []string{"test"},
			testBench:   &test.TestBench{},
			expected:    []string{"test"},
		},
	}

	for _, testdata := range tests {
		testutil.Run(t, testdata.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, test.MockK8sClient)
			t.Override(&newStatusCheck, func(status.Config, *label.DefaultLabeller) status.Checker {
				return dummyStatusChecker{}
			})

			runner := MockRunnerV1(t, testdata.testBench, nil, []*latest_v1.Artifact{{ImageName: "img1"},
				{ImageName: "img2"}}, nil)
			runner.RunCtx.Namespaces = testdata.Namespaces

			runner.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})

			t.CheckDeepEqual(testdata.expected, runner.RunCtx.GetNamespaces())
		})
	}
}

func TestSkaffoldDeployRenderOnly(t *testing.T) {
	testutil.Run(t, "does not make kubectl calls", func(t *testutil.T) {
		runCtx := &runcontext.RunContext{
			Opts: config.SkaffoldOptions{
				Namespace:  "testNamespace",
				RenderOnly: true,
			},
			KubeContext: "does-not-exist",
		}

		deployer, err := getDeployer(runCtx, nil)
		t.RequireNoError(err)
		r := SkaffoldRunner{
			RunCtx:     runCtx,
			kubectlCLI: kubectl.NewCLI(runCtx, ""),
			Deployer:   deployer,
		}
		var builds []graph.Artifact

		err = r.Deploy(context.Background(), ioutil.Discard, builds)

		t.CheckNoError(err)
	})
}

type dummyStatusChecker struct{}

func (d dummyStatusChecker) Check(_ context.Context, _ io.Writer) error {
	return nil
}
