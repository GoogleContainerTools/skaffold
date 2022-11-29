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

package kubectl

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestCLI(t *testing.T) {
	const (
		kubeContext = "some-kubecontext"
		output      = "this is the expected output"
	)

	tests := []struct {
		name            string
		kubeconfig      string
		namespace       string
		expectedCommand string
	}{
		{
			name:            "without namespace or kubeconfig",
			expectedCommand: "kubectl --context some-kubecontext exec arg1 arg2",
		},
		{
			name:            "only namespace, no kubeconfig",
			namespace:       "some-namespace",
			expectedCommand: "kubectl --context some-kubecontext --namespace some-namespace exec arg1 arg2",
		},
		{
			name:            "only kubeconfig, no namespace",
			kubeconfig:      "some-kubeconfig",
			expectedCommand: "kubectl --context some-kubecontext --kubeconfig some-kubeconfig exec arg1 arg2",
		},
		{
			name:            "with namespace and kubeconfig",
			kubeconfig:      "some-kubeconfig",
			namespace:       "some-namespace",
			expectedCommand: "kubectl --context some-kubecontext --namespace some-namespace --kubeconfig some-kubeconfig exec arg1 arg2",
		},
	}

	// test cli.Run()
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRun(
				test.expectedCommand,
			))

			cli := NewCLI(&mockConfig{
				kubeContext: kubeContext,
				kubeConfig:  test.kubeconfig,
				namespace:   test.namespace,
			}, "")
			err := cli.Run(context.Background(), nil, nil, "exec", "arg1", "arg2")

			t.CheckNoError(err)
		})
	}

	// test cli.RunOut()
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.expectedCommand,
				output,
			))

			cli := NewCLI(&mockConfig{
				kubeContext: kubeContext,
				kubeConfig:  test.kubeconfig,
				namespace:   test.namespace,
			}, "")
			out, err := cli.RunOut(context.Background(), "exec", "arg1", "arg2")

			t.CheckNoError(err)
			t.CheckDeepEqual(string(out), output)
		})
	}

	// test cli.CommandWithStrictCancellation()
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.expectedCommand,
				output,
			))

			cli := NewCLI(&mockConfig{
				kubeContext: kubeContext,
				kubeConfig:  test.kubeconfig,
				namespace:   test.namespace,
			}, "")
			cmd := cli.CommandWithStrictCancellation(context.Background(), "exec", "arg1", "arg2")
			out, err := util.RunCmdOut(context.Background(), cmd.Cmd)

			t.CheckNoError(err)
			t.CheckDeepEqual(string(out), output)
		})
	}
}

type mockConfig struct {
	kubeContext string
	kubeConfig  string
	namespace   string
}

func (c *mockConfig) GetKubeContext() string   { return c.kubeContext }
func (c *mockConfig) GetKubeConfig() string    { return c.kubeConfig }
func (c *mockConfig) GetKubeNamespace() string { return c.namespace }
