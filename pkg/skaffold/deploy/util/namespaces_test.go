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

	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
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

func TestGetAllPodNamespaces(t *testing.T) {
	tests := []struct {
		description  string
		ns           string
		helmReleases []latest.HelmRelease
		apiConfig    api.Config
		env          []string
		expected     []string
		shouldErr    bool
	}{
		{
			description: "current config empty, ns empty with helm releases",
			ns:          "",
			apiConfig:   api.Config{CurrentContext: ""},
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
			expected: []string{"", "bar", "baz", "foo"},
		},
		{
			description: "current config empty, ns empty",
			ns:          "",
			apiConfig:   api.Config{CurrentContext: ""},
			expected:    []string{""},
		},
		{
			description: "ns empty, current config set",
			ns:          "",
			apiConfig: api.Config{CurrentContext: "test",
				Contexts: map[string]*api.Context{
					"test": {Namespace: "test-ns"}}},
			expected: []string{"test-ns"},
		},
		{
			description: "ns set and current config set",
			ns:          "cli-ns",
			apiConfig: api.Config{CurrentContext: "test",
				Contexts: map[string]*api.Context{
					"test": {Namespace: "test-ns"}}},
			expected: []string{"cli-ns"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.env })
			t.Override(&kubectx.CurrentConfig, func() (api.Config, error) {
				return test.apiConfig, nil
			})
			ns, err := GetAllPodNamespaces(test.ns, []latest.Pipeline{
				{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
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
