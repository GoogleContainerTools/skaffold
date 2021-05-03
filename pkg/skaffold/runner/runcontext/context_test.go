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

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRunContext_UpdateNamespaces(t *testing.T) {
	tests := []struct {
		description   string
		oldNamespaces []string
		newNamespaces []string
		expected      []string
	}{
		{
			description:   "update namespace when not present in runContext",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{"another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespace with duplicates should not return duplicate",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{"another", "foo", "another"},
			expected:      []string{"another", "foo", "test"},
		},
		{
			description:   "update namespaces when namespaces is empty",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{},
			expected:      []string{"test", "foo"},
		},
		{
			description:   "update namespaces when runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{"test", "another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespaces when both namespaces and runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{},
			expected:      []string{},
		},
		{
			description:   "update namespace when runcontext namespace has an empty string",
			oldNamespaces: []string{""},
			newNamespaces: []string{"another"},
			expected:      []string{"another"},
		},
		{
			description:   "update namespace when namespace is empty string",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{""},
			expected:      []string{"test"},
		},
		{
			description:   "update namespace when namespace is empty string and runContext is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{""},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runCtx := &RunContext{
				Namespaces: test.oldNamespaces,
			}

			runCtx.UpdateNamespaces(test.newNamespaces)

			t.CheckDeepEqual(test.expected, runCtx.Namespaces)
		})
	}
}

func TestPipelines_StatusCheck(t *testing.T) {
	tests := []struct {
		description   string
		statusCheckP1 *bool
		statusCheckP2 *bool
		wantEnabled   *bool
		wantErr       bool
	}{
		{
			description: "both pipelines' statusCheck values are unspecified",
			wantEnabled: util.BoolPtr(true),
		},
		{
			description:   "one pipeline's statusCheck value is true",
			statusCheckP1: util.BoolPtr(true),
			wantEnabled:   util.BoolPtr(true),
		},
		{
			description:   "one pipeline's statusCheck value is false",
			statusCheckP1: util.BoolPtr(false),
			wantEnabled:   util.BoolPtr(false),
		},
		{
			description:   "one pipeline's statusCheck value is true, one pipeline's statusCheck value is false",
			statusCheckP1: util.BoolPtr(true),
			statusCheckP2: util.BoolPtr(false),
			wantErr:       true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			p := &Pipelines{
				pipelines: []latest_v1.Pipeline{
					{
						Deploy: latest_v1.DeployConfig{
							StatusCheck: test.statusCheckP1,
						},
					},
					{
						Deploy: latest_v1.DeployConfig{
							StatusCheck: test.statusCheckP2,
						},
					},
				},
			}
			gotEnabled, err := p.StatusCheck()
			if err != nil && !test.wantErr {
				t.Errorf("p.StatusCheck() got error %v, want no error", err)
			}
			if err == nil && test.wantErr {
				t.Errorf("p.StatusCheck() got no error, want error")
			}
			t.CheckDeepEqual(test.wantEnabled, gotEnabled)
		})
	}
}
