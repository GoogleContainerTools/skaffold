/*
Copyright 2020 The Skaffold Authors

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

package runner

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployer(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.DeployType
		expected    deploy.Deployer
	}{
		{
			description: "no deployer",
			expected:    deploy.DeployerMux{},
		},
		{
			description: "helm deployer",
			cfg:         latest.DeployType{HelmDeploy: &latest.HelmDeploy{}},
			expected:    deploy.NewHelmDeployer(&runcontext.RunContext{}, nil),
		},
		{
			description: "kubectl deployer",
			cfg:         latest.DeployType{KubectlDeploy: &latest.KubectlDeploy{}},
			expected: deploy.NewKubectlDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Flags: latest.KubectlFlags{},
							},
						},
					},
				},
			}, nil),
		},
		{
			description: "kustomize deployer",
			cfg:         latest.DeployType{KustomizeDeploy: &latest.KustomizeDeploy{}},
			expected: deploy.NewKustomizeDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								Flags: latest.KubectlFlags{},
							},
						},
					},
				},
			}, nil),
		},
		{
			description: "kpt deployer",
			cfg:         latest.DeployType{KptDeploy: &latest.KptDeploy{}},
			expected:    deploy.NewKptDeployer(&runcontext.RunContext{}, nil),
		},
		{
			description: "multiple deployers",
			cfg: latest.DeployType{
				HelmDeploy: &latest.HelmDeploy{},
				KptDeploy:  &latest.KptDeploy{},
			},
			expected: deploy.DeployerMux{
				deploy.NewHelmDeployer(&runcontext.RunContext{}, nil),
				deploy.NewKptDeployer(&runcontext.RunContext{}, nil),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deployer := getDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: test.cfg,
					},
				},
			}, nil)

			t.CheckTypeEquality(test.expected, deployer)

			if reflect.TypeOf(test.expected) == reflect.TypeOf(deploy.DeployerMux{}) {
				expected := test.expected.(deploy.DeployerMux)
				deployers := deployer.(deploy.DeployerMux)
				t.CheckDeepEqual(len(expected), len(deployers))
				for i, v := range expected {
					t.CheckTypeEquality(v, deployers[i])
				}
			}
		})
	}
}

func TestCreateComponents(t *testing.T) {
	gitExample, _ := tag.NewGitCommit("", "")
	envExample, _ := tag.NewEnvTemplateTagger("test")

	tests := []struct {
		description          string
		customTemplateTagger *latest.CustomTemplateTagger
		expected             map[string]tag.Tagger
		shouldErr            bool
	}{
		{
			description: "correct component types",
			customTemplateTagger: &latest.CustomTemplateTagger{
				Components: []latest.TaggerComponent{
					{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
					{Name: "FOE", Component: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}}},
					{Name: "BAR", Component: latest.TagPolicy{EnvTemplateTagger: &latest.EnvTemplateTagger{Template: "test"}}},
					{Name: "BAT", Component: latest.TagPolicy{DateTimeTagger: &latest.DateTimeTagger{}}},
				},
			},
			expected: map[string]tag.Tagger{
				"FOO": gitExample,
				"FOE": &tag.ChecksumTagger{},
				"BAR": envExample,
				"BAT": tag.NewDateTimeTagger("", ""),
			},
		},
		{
			description: "customTemplate is an invalid component",
			customTemplateTagger: &latest.CustomTemplateTagger{
				Components: []latest.TaggerComponent{
					{Name: "FOO", Component: latest.TagPolicy{CustomTemplateTagger: &latest.CustomTemplateTagger{Template: "test"}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "recurring names",
			customTemplateTagger: &latest.CustomTemplateTagger{
				Components: []latest.TaggerComponent{
					{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
					{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown component",
			customTemplateTagger: &latest.CustomTemplateTagger{
				Components: []latest.TaggerComponent{
					{Name: "FOO", Component: latest.TagPolicy{}},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			components, err := CreateComponents(test.customTemplateTagger)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, len(test.expected), len(components))
			for k, v := range test.expected {
				t.CheckTypeEquality(v, components[k])
			}
		})
	}
}
