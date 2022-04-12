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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCollectHelmReleasesNamespaces(t *testing.T) {
	tests := []struct {
		description  string
		helmReleases []latest.HelmRelease
		env          []string
		expected     []string
		shouldErr    bool
	}{
		{
			description: "namspaces are collected correctly",
			helmReleases: []latest.HelmRelease{
				{
					Namespace: "foo",
				},
				{
					Namespace: "bar",
				},
				{
					Namespace: "baz",
				},
			},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			description: "namespaces are collected correctly with env expansion",
			helmReleases: []latest.HelmRelease{
				{
					Namespace: "{{.FOO}}",
				},
				{
					Namespace: "bar",
				},
				{
					Namespace: "baz",
				},
			},
			env:      []string{"FOO=foo"},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			description: "should error when template expansion fails",
			helmReleases: []latest.HelmRelease{
				{
					Namespace: "{{.DOESNT_EXIST_AND_SHOULD_ERROR_AS_SUCH}}",
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.env })
			ns, err := collectHelmReleasesNamespaces([]latest.Pipeline{
				{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							HelmDeploy: &latest.HelmDeploy{
								Releases: test.helmReleases,
							},
						},
					},
				},
			})
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(test.expected, ns)
			}
		})
	}
}
