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

package util

import (
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	latestV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetAllPodNamespaces(t *testing.T) {
	tests := []struct {
		description    string
		argNamespace   string
		currentContext string
		cfg            latestV2.Pipeline
		expected       []string
	}{
		{
			description:  "namespace provided on the command line",
			argNamespace: "ns",
			expected:     []string{"ns"},
		},
		{
			description:    "kube context's namespace",
			currentContext: "prod-context",
			expected:       []string{"prod"},
		},
		{
			description:    "default namespace",
			currentContext: "unknown context",
			expected:       []string{""},
		},
		{
			description:  "add namespaces for helm",
			argNamespace: "ns",
			cfg: latestV2.Pipeline{
				Deploy: latestV2.DeployConfig{
					DeployType: latestV2.DeployType{
						HelmDeploy: &latestV2.HelmDeploy{
							Releases: []latestV2.HelmRelease{
								{Namespace: "ns3"},
								{Namespace: ""},
								{Namespace: ""},
								{Namespace: "ns2"},
							},
						},
					},
				},
			},
			expected: []string{"ns", "ns2", "ns3"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			t.Override(&context.CurrentConfig, func() (api.Config, error) {
				return api.Config{
					CurrentContext: test.currentContext,
					Contexts: map[string]*api.Context{
						"prod-context": {Namespace: "prod"},
					},
				}, nil
			})

			namespaces, err := GetAllPodNamespaces(test.argNamespace, []latestV2.Pipeline{test.cfg})

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, namespaces)
		})
	}
}
