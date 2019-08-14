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

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRunContext_UpdateNamespaces(t *testing.T) {
	tests := []struct {
		description string
		runContext  *RunContext
		namespaces  []string
		expected    []string
	}{
		{
			description: "update namespace when not present in runContext",
			runContext:  &RunContext{Namespaces: []string{"test"}},
			namespaces:  []string{"another"},
			expected:    []string{"another", "test"},
		},
		{
			description: "update namespace with duplicates should not return duplicate",
			runContext:  &RunContext{Namespaces: []string{"test", "foo"}},
			namespaces:  []string{"another", "foo", "another"},
			expected:    []string{"another", "foo", "test"},
		},
		{
			description: "update namespaces when namespaces is empty",
			runContext:  &RunContext{Namespaces: []string{"test", "foo"}},
			namespaces:  []string{},
			expected:    []string{"test", "foo"},
		},
		{
			description: "update namespaces when runcontext namespaces is empty",
			runContext:  &RunContext{Namespaces: []string{}},
			namespaces:  []string{"test", "another"},
			expected:    []string{"another", "test"},
		},
		{
			description: "update namespaces when both namespaces and runcontext namespaces is empty",
			runContext:  &RunContext{Namespaces: []string{}},
			namespaces:  []string{},
			expected:    []string{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			test.runContext.UpdateNamespaces(test.namespaces)
			t.CheckDeepEqual(test.expected, test.runContext.Namespaces)
		})
	}
}
