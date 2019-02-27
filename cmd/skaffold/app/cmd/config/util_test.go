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

package config

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestGetKubeContext(t *testing.T) {
	tests := []struct {
		description   string
		kubeContext   string
		schemaContext string
		expected      string
		shouldErr     bool
	}{
		{
			description:   "kubectl current context",
			kubeContext:   "kubectl-current-context",
			schemaContext: "",
			expected:      "kubectl-current-context",
		},
		{
			description:   "schema context",
			kubeContext:   "kubectl-current-context",
			schemaContext: "schema-context",
			expected:      "schema-context",
		},
	}

	for _, test := range tests {
		if test.kubeContext != "" {
			restore := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: test.kubeContext})
			defer restore()
		}

		actual, err := GetKubeContext(test.schemaContext)
		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
	}
}
