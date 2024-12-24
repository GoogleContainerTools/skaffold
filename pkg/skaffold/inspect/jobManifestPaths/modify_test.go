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
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPrintJobManifestPathsModify(t *testing.T) {
	tests := []struct {
		description string
		input       string
		shouldErr   bool
		expected    string
		originalCfg latest.SkaffoldConfig
	}{
		{
			description: "successfully modifies jobManifestPath in verify stanza as intended",
			input:       "{\"verifyJobManifestPaths\":{\"foo\":\"modified-foo.yaml\"},\"customActionJobManifestPaths\":{}}",
			expected: `apiVersion: skaffold/v4beta5
kind: Config
verify:
  - name: foo
    container:
      name: foo
      image: foo
      env:
        - name: key
          value: value
    executionMode:
      kubernetesCluster:
        jobManifestPath: modified-foo.yaml
`,
			originalCfg: latest.SkaffoldConfig{
				APIVersion: "skaffold/v4beta5",
				Kind:       "Config",
				Pipeline: latest.Pipeline{
					Verify: []*latest.VerifyTestCase{
						{
							Name: "foo",
							Container: latest.VerifyContainer{
								Name:  "foo",
								Image: "foo",
								Env: []latest.VerifyEnvVar{
									{
										Name:  "key",
										Value: "value",
									},
								},
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
			},
		},
		{
			description: "failure with invalid transform yaml input",
			input:       "invalid",
			shouldErr:   true,
		},
		{
			description: "successfully modifies jobManifestPath in customActions stanza as intended",
			input:       `{"verifyJobManifestPaths":{},"customActionJobManifestPaths":{"action1":"modified-manifest.yaml"}}`,
			expected: `apiVersion: skaffold/v4beta5
kind: Config
verify:
  - name: foo
    container:
      name: foo
      image: foo
    executionMode:
      kubernetesCluster:
        jobManifestPath: verify-manifest.yaml
customActions:
  - name: action1
    executionMode:
      kubernetesCluster:
        jobManifestPath: modified-manifest.yaml
    containers:
      - name: task1
        image: task1-img
  - name: action2
    executionMode:
      kubernetesCluster:
        jobManifestPath: custom-action-job-manifest.yaml
    containers:
      - name: task2
        image: task2-img
`,
			originalCfg: latest.SkaffoldConfig{
				APIVersion: "skaffold/v4beta5",
				Kind:       "Config",
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
										JobManifestPath: "verify-manifest.yaml",
									},
								},
							},
						},
					},
					CustomActions: []latest.Action{
						{
							Name: "action1",
							Containers: []latest.VerifyContainer{
								{
									Name:  "task1",
									Image: "task1-img",
								},
							},
							ExecutionModeConfig: latest.ActionExecutionModeConfig{
								VerifyExecutionModeType: latest.VerifyExecutionModeType{
									KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
										JobManifestPath: "original-manifest.yaml",
									},
								},
							},
						},
						{
							Name: "action2",
							Containers: []latest.VerifyContainer{
								{
									Name:  "task2",
									Image: "task2-img",
								},
							},
							ExecutionModeConfig: latest.ActionExecutionModeConfig{
								VerifyExecutionModeType: latest.VerifyExecutionModeType{
									KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
										JobManifestPath: "custom-action-job-manifest.yaml",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "successfully modifies jobManifestPath in customActions and verify stanza as intended",
			input:       `{"verifyJobManifestPaths":{"foo":"modified-foo.yaml"},"customActionJobManifestPaths":{"action1":"modified-manifest.yaml"}}`,
			expected: `apiVersion: skaffold/v4beta5
kind: Config
verify:
  - name: foo
    container:
      name: foo
      image: foo
    executionMode:
      kubernetesCluster:
        jobManifestPath: modified-foo.yaml
customActions:
  - name: action1
    executionMode:
      kubernetesCluster:
        jobManifestPath: modified-manifest.yaml
    containers:
      - name: task1
        image: task1-img
`,
			originalCfg: latest.SkaffoldConfig{
				APIVersion: "skaffold/v4beta5",
				Kind:       "Config",
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
										JobManifestPath: "verify-manifest.yaml",
									},
								},
							},
						},
					},
					CustomActions: []latest.Action{
						{
							Name: "action1",
							Containers: []latest.VerifyContainer{
								{
									Name:  "task1",
									Image: "task1-img",
								},
							},
							ExecutionModeConfig: latest.ActionExecutionModeConfig{
								VerifyExecutionModeType: latest.VerifyExecutionModeType{
									KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{
										JobManifestPath: "original-manifest.yaml",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			inputFile := t.TempFile("", []byte(test.input))

			t.Override(&getCfgs, func(context.Context, config.SkaffoldOptions) ([]util.VersionedConfig, error) {
				return []util.VersionedConfig{&test.originalCfg}, nil
			})
			var b bytes.Buffer
			err := Modify(context.Background(), &b, config.SkaffoldOptions{}, inputFile, "")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, b.String())
		})
	}
}
