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

package deploy

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployments(t *testing.T) {

	var tests = []struct {
		description string
		command     util.Command
		expected    []string
		shouldErr   bool
	}{
		{
			description: "returns deployments",
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace test get deployments --output jsonpath='{.items[*].metadata.name}'", "dep1 dep2"),
			expected: []string{"dep1", "dep2"},
		},
		{
			description: "no deployments",
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl --context kubecontext --namespace test get deployments --output jsonpath='{.items[*].metadata.name}'", ""),
			expected: []string{},
		},
		{
			description: "get deployments error",
			command: testutil.NewFakeCmd(t).
				WithRunOutErr("kubectl --context kubecontext --namespace test get deployments --output jsonpath='{.items[*].metadata.name}'", "", fmt.Errorf("error")),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			reset := testutil.Override(t, &util.DefaultExecCommand, test.command)
			defer reset()
			cli := kubectl.CLI{
				Namespace:   "test",
				KubeContext: testKubeContext,
			}
			actual, err := getDeployments(context.Background(), cli)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}
