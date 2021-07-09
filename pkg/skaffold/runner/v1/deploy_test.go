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
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeploy(t *testing.T) {
	tests := []struct {
		description string
		testBench   *TestBench
		shouldErr   bool
	}{
		{
			description: "deploy succeeds",
			testBench:   &TestBench{},
		},
		{
			description: "deploy fails",
			testBench:   &TestBench{deployErrors: []error{errors.New("deploy error")}},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)

			r := createRunner(t, test.testBench, nil, []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}, nil)
			out := new(bytes.Buffer)

			err := r.Deploy(context.Background(), out, []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestDeployNamespace(t *testing.T) {
	tests := []struct {
		description string
		Namespaces  []string
		testBench   *TestBench
		expected    []string
	}{
		{
			description: "deploy shd add all namespaces to run Context",
			Namespaces:  []string{"test", "test-ns"},
			testBench:   NewTestBench().WithDeployNamespaces([]string{"test-ns", "test-ns-1"}),
			expected:    []string{"test", "test-ns", "test-ns-1"},
		},
		{
			description: "deploy without command opts namespace",
			testBench:   NewTestBench().WithDeployNamespaces([]string{"test-ns", "test-ns-1"}),
			expected:    []string{"test-ns", "test-ns-1"},
		},
		{
			description: "deploy with no namespaces returned",
			Namespaces:  []string{"test"},
			testBench:   &TestBench{},
			expected:    []string{"test"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})
			t.Override(&client.Client, mockK8sClient)

			r := createRunner(t, test.testBench, nil, []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}, nil)
			r.runCtx.Namespaces = test.Namespaces

			err := r.Deploy(context.Background(), ioutil.Discard, []graph.Artifact{
				{ImageName: "img1", Tag: "img1:tag1"},
				{ImageName: "img2", Tag: "img2:tag2"},
			})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, r.runCtx.GetNamespaces())
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

		deployer, err := runner.GetDeployer(runCtx, component.NoopComponentProvider{}, nil)
		t.RequireNoError(err)
		r := SkaffoldRunner{
			runCtx:     runCtx,
			kubectlCLI: kubectl.NewCLI(runCtx, ""),
			deployer:   deployer,
		}
		var builds []graph.Artifact

		err = r.Deploy(context.Background(), ioutil.Discard, builds)

		t.CheckNoError(err)
	})
}
