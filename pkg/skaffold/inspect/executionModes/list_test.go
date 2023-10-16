/*
Copyright 2021 The Skaffold Authors

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

package inspect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPrintExecutionModesList(t *testing.T) {
	tests := []struct {
		description string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all executionModes where no executionMode is set in the verify config",
			expected:    "{\"verifyExecutionModes\":{},\"customActionsExecutionModes\":{}}" + "\n",
			module:      []string{"cfg-without-executionModes"},
		},
		{
			description: "print all executionModes where one executionMode is set in the verify and customAction config via a profile but no customAction arg",
			expected:    "{\"verifyExecutionModes\":{\"foo\":\"kubernetesCluster\"},\"customActionsExecutionModes\":{}}" + "\n",
			profiles:    []string{"has-verify-executionMode"},
			module:      []string{"cfg-without-executionModes"},
		},
		{
			description: "print all executionModes where one executionMode is set in the verify config via a module",
			expected:    "{\"verifyExecutionModes\":{\"bar\":\"kubernetesCluster\"},\"customActionsExecutionModes\":{}}" + "\n",
			module:      []string{"cfg-with-executionModes"},
		},
		{
			description: "actionable error",
			err:         sErrors.MainConfigFileNotFoundErr("path/to/skaffold.yaml", fmt.Errorf("failed to read file : %q", "skaffold.yaml")),
			expected:    `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/skaffold.yaml\": failed to read file : \"skaffold.yaml\". Check that the specified configuration file exists at \"path/to/skaffold.yaml\"."}` + "\n",
		},
		{
			description: "generic error",
			err:         errors.New("some error occurred"),
			expected:    `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
		{
			description: "print executionModes for custom actions",
			expected:    `{"verifyExecutionModes":{},"customActionsExecutionModes":{"action1":"kubernetesCluster","action2":"local"}}` + "\n",
			module:      []string{"cfg-with-customActions"},
		},
		{
			description: "print executionModes for custom action and verify",
			expected:    `{"verifyExecutionModes":{"bar":"kubernetesCluster"},"customActionsExecutionModes":{"action1":"kubernetesCluster"}}` + "\n",
			module:      []string{"cfg-with-customActions-and-verify"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-without-executionModes"},
					Pipeline: latest.Pipeline{},
					Profiles: []latest.Profile{
						{Name: "has-verify-executionMode",
							Pipeline: latest.Pipeline{
								Verify: []*latest.VerifyTestCase{
									{
										Name: "foo",
										Container: latest.VerifyContainer{
											Name:  "foo",
											Image: "foo",
										},
										ExecutionMode: latest.VerifyExecutionModeConfig{
											VerifyExecutionModeType: latest.VerifyExecutionModeType{
												KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
											},
										},
									},
								},
							},
						}},
				}, SourceFile: "path/to/cfg-without-executionModes"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-with-executionModes"},
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{
								Name: "bar",
								Container: latest.VerifyContainer{
									Name:  "bar",
									Image: "bar",
								},
								ExecutionMode: latest.VerifyExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
									},
								},
							},
						},
					},
				}, SourceFile: "path/to/cfg-with-default-namespace"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-with-customActions"},
					Pipeline: latest.Pipeline{
						CustomActions: []latest.Action{
							{
								Name: "action1",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
									},
								},
								Containers: []latest.VerifyContainer{
									{
										Name:  "task1",
										Image: "task1-image",
									},
								},
							},
							{
								Name: "action2",
								Containers: []latest.VerifyContainer{
									{
										Name:  "task2",
										Image: "task2-image",
									},
								},
							},
						},
					},
				}, SourceFile: "path/to/cfg-with-customActions"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-with-customActions-and-verify"},
					Pipeline: latest.Pipeline{
						Verify: []*latest.VerifyTestCase{
							{
								Name: "bar",
								Container: latest.VerifyContainer{
									Name:  "bar",
									Image: "bar",
								},
								ExecutionMode: latest.VerifyExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
									},
								},
							},
						},
						CustomActions: []latest.Action{
							{
								Name: "action1",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
									},
								},
								Containers: []latest.VerifyContainer{
									{
										Name:  "task1",
										Image: "task1-image",
									},
								},
							},
						},
					},
				}, SourceFile: "path/to/cfg-with-customActions-and-verify"},
			}
			t.Override(&inspect.GetConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Verify = profile.Verify
							c.CustomActions = profile.CustomActions
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintExecutionModesList(context.Background(), &buf, inspect.Options{
				OutFormat: "json", Modules: test.module, Profiles: test.profiles})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
