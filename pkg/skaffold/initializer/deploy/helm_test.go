/*
Copyright 2022 The Skaffold Authors

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

package deploy

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeployConfig(t *testing.T) {
	tests := []struct {
		description string
		input       map[string][]string
		expected    []latest.HelmRelease
		helm        helm
		expected    []latest.HelmRelease
	}{
		{
			description: "charts with one or more values file",
			input: map[string][]string{
				"charts":     {"charts/val.yml", "charts/values.yaml"},
				"charts-foo": {"charts-foo/values.yaml"},
			},
			expected: []latest.HelmRelease{
				{
					Name:        "charts-foo",
					ChartPath:   "charts-foo",
					ValuesFiles: []string{"charts-foo/values.yaml"},
				},
				{
					Name:        "charts",
					ChartPath:   "charts",
					ValuesFiles: []string{"charts/val.yml", "charts/values.yaml"},
				},
			},
			description: "charts with one or more values file",
			helm: newHelmInitializer(
				map[string][]string{
					"charts":     {"charts/val.yml", "charts/values.yaml"},
					"charts-foo": {"charts-foo/values.yaml"},
				}),
			expected: []latest.HelmRelease{
				{
					Name:        "charts",
					ChartPath:   "charts",
					ValuesFiles: []string{"charts/val.yml", "charts/values.yaml"},
				},
				{
					Name:        "charts-foo",
					ChartPath:   "charts-foo",
					ValuesFiles: []string{"charts-foo/values.yaml"},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&readFile, func(_ string) ([]byte, error) {
				return []byte{}, nil
			})
			h := newHelmInitializer(test.input)
			d, _ := h.DeployConfig()
			CheckHelmInitStruct(t, test.expected, d.LegacyHelmDeploy.Releases)
			overrides, err := parseImagesFromReader(test.contents, "dummy.yaml")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.overrides, overrides)
			d, _ := test.helm.DeployConfig()
			t.CheckDeepEqual(test.expected, d.HelmDeploy.Releases)
		})
	}
}
