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

func TestPrintJobManifestPathsList(t *testing.T) {
	tests := []struct {
		description string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all jobManifestPaths where no jobManifestPath is set in the verify config",
			expected:    "{\"verifyJobManifestPaths\":{},\"customActionJobManifestPaths\":{}}" + "\n",
			module:      []string{"cfg-without-jobManifestPaths"},
		},
		{
			description: "print all jobManifestPaths where one jobManifestPath is set in the verify config via a profile",
			expected:    "{\"verifyJobManifestPaths\":{\"foo\":\"foo.yaml\"},\"customActionJobManifestPaths\":{}}" + "\n",
			profiles:    []string{"has-jobManifestPath"},
			module:      []string{"cfg-without-jobManifestPaths"},
		},
		{
			description: "print all jobManifestPaths where one jobManifestPath is set in the verify config via a module",
			expected:    "{\"verifyJobManifestPaths\":{\"bar\":\"bar.yaml\"},\"customActionJobManifestPaths\":{}}" + "\n",
			module:      []string{"cfg-with-jobManifestPaths"},
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
			description: "print all jobManifestPaths where one jobManifestPath is set in the customActions config",
			expected:    `{"verifyJobManifestPaths":{},"customActionJobManifestPaths":{"action1":"./path/to/manifest.yaml"}}` + "\n",
			module:      []string{"cfg-customActions-with-jobManifestPaths"},
		},
		{
			description: "print all jobManifestPaths where jobManifestPaths are set in customActions and verify config",
			expected:    `{"verifyJobManifestPaths":{"bar":"bar.yaml"},"customActionJobManifestPaths":{"action1":"./path/to/manifest.yaml"}}` + "\n",
			module:      []string{"cfg-customActions-and-verify-with-jobManifestPaths"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-without-jobManifestPaths"},
					Pipeline: latest.Pipeline{},
					Profiles: []latest.Profile{
						{Name: "has-jobManifestPath",
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
												KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
													JobManifestPath: "foo.yaml",
												},
											},
										},
									},
								},
							},
						}},
				}, SourceFile: "path/to/cfg-without-jobManifestPaths"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-with-jobManifestPaths"},
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
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
											JobManifestPath: "bar.yaml",
										},
									},
								},
							},
						},
					},
				}, SourceFile: "path/to/cfg-with-default-namespace"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-customActions-with-jobManifestPaths"},
					Pipeline: latest.Pipeline{
						CustomActions: []latest.Action{
							{
								Name: "action1",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
											JobManifestPath: "./path/to/manifest.yaml",
										},
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
				}, SourceFile: "path/to/cfg-customActions-with-jobManifestPaths"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-customActions-and-verify-with-jobManifestPaths"},
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
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
											JobManifestPath: "bar.yaml",
										},
									},
								},
							},
						},
						CustomActions: []latest.Action{
							{
								Name: "action1",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
											JobManifestPath: "./path/to/manifest.yaml",
										},
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
				}, SourceFile: "path/to/cfg-customActions-and-verify-with-jobManifestPaths"},
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
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintJobManifestPathsList(context.Background(), &buf, inspect.Options{
				OutFormat: "json", Modules: test.module, Profiles: test.profiles})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
