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

package runcontext

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name             string
		kubeContext      string
		kubeConfig       string
		namespace        string
		defaultNamespace *string
		command          string
		output           string
		expected         string
	}{
		{
			name:     "current context",
			command:  "kubectl config view --minify -o jsonpath='{..namespace}'",
			output:   "'current-namespace'",
			expected: "current-namespace",
		},
		{
			name:        "selected context overrides current context",
			kubeContext: "ee-lte-1.codemowers.io/demo",
			command:     "kubectl --context ee-lte-1.codemowers.io/demo config view --minify -o jsonpath='{..namespace}'",
			output:      "'demo'",
			expected:    "demo",
		},
		{
			name:       "explicit kubeconfig",
			kubeConfig: "/tmp/other-kubeconfig",
			command:    "kubectl --kubeconfig /tmp/other-kubeconfig config view --minify -o jsonpath='{..namespace}'",
			output:     "'other-namespace'",
			expected:   "other-namespace",
		},
		{
			name:        "selected context and explicit kubeconfig",
			kubeContext: "ee-lte-1.codemowers.io/demo",
			kubeConfig:  "/tmp/other-kubeconfig",
			command:     "kubectl --context ee-lte-1.codemowers.io/demo --kubeconfig /tmp/other-kubeconfig config view --minify -o jsonpath='{..namespace}'",
			output:      "'demo'",
			expected:    "demo",
		},
		{
			name:        "context without a namespace",
			kubeContext: "no-namespace",
			command:     "kubectl --context no-namespace config view --minify -o jsonpath='{..namespace}'",
			output:      "''",
		},
		{
			name:             "explicit namespace takes precedence",
			kubeContext:      "ee-lte-1.codemowers.io/demo",
			namespace:        "override",
			defaultNamespace: util.Ptr("configured"),
			expected:         "override",
		},
		{
			name:             "configured default namespace takes precedence",
			kubeContext:      "ee-lte-1.codemowers.io/demo",
			defaultNamespace: util.Ptr("configured"),
			expected:         "configured",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			// A lookup of the current context must not satisfy a selected-context lookup.
			commands := testutil.CmdRunOutOnce(test.command, test.output)
			t.Override(&util.DefaultExecCommand, commands)
			rc := RunContext{
				KubeContext: test.kubeContext,
				Opts: config.SkaffoldOptions{
					KubeConfig: test.kubeConfig,
					Namespace:  test.namespace,
				},
				Pipelines: NewPipelines(map[string]latest.Pipeline{
					"app": {Deploy: latest.DeployConfig{DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{DefaultNamespace: test.defaultNamespace},
					}}},
				}, []string{"app"}),
			}
			t.CheckDeepEqual(test.expected, rc.GetNamespace())
		})
	}
}
