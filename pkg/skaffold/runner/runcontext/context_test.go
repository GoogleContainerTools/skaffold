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
		expected         string
	}{
		{
			name:     "current context",
			expected: "current-namespace",
		},
		{
			name:        "selected context overrides current context",
			kubeContext: "ee-lte-1.codemowers.io/demo",
			expected:    "demo",
		},
		{
			name:       "explicit kubeconfig",
			kubeConfig: "/tmp/other-kubeconfig",
			expected:   "other-namespace",
		},
		{
			name:        "selected context and explicit kubeconfig",
			kubeContext: "ee-lte-1.codemowers.io/demo",
			kubeConfig:  "/tmp/other-kubeconfig",
			expected:    "demo",
		},
		{
			name:        "context without a namespace",
			kubeContext: "no-namespace",
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
			current := t.TempFile("config", []byte(`apiVersion: v1
kind: Config
current-context: current
contexts:
- name: current
  context:
    namespace: current-namespace
- name: ee-lte-1.codemowers.io/demo
  context:
    namespace: demo
- name: no-namespace
  context: {}
`))
			t.Setenv("KUBECONFIG", current)
			if test.kubeConfig != "" {
				test.kubeConfig = t.TempFile("other-config", []byte(`apiVersion: v1
kind: Config
current-context: other
contexts:
- name: other
  context:
    namespace: other-namespace
- name: ee-lte-1.codemowers.io/demo
  context:
    namespace: demo
`))
			}
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
