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
)

func TestResolveKubectlContext(t *testing.T) {
	tests := []struct {
		description   string
		cmdContext    string
		schemaContext string
		expected      string
	}{
		{
			description:   "resolve command context",
			cmdContext:    "cmd-context",
			schemaContext: "schema-context",
			expected:      "cmd-context",
		},
		{
			description:   "resolve schema context",
			schemaContext: "schema-context",
			expected:      "schema-context",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.cmdContext != "" {
				kubecontext = test.cmdContext
				defer func() { kubecontext = "" }()
			}

			ResolveKubectlContext(test.schemaContext)
			testutil.CheckDeepEqual(t, test.expected, kubecontext)
		})
	}
}
