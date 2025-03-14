/*
Copyright 2025 The Skaffold Authors

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

package helm

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDepBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		chartPath   string
		flags       latest.HelmDeployFlags
		expected    []string
	}{
		{
			description: "basic",
			chartPath:   "chart/path",
			expected:    []string{"dep", "build", "chart/path"},
		},
		{
			description: "with flags",
			chartPath:   "chart/path",
			flags: latest.HelmDeployFlags{
				DepBuild: []string{"--skip-refresh"},
			},
			expected: []string{"dep", "build", "chart/path", "--skip-refresh"},
		},
		{
			description: "with multiple flags",
			chartPath:   "chart/path",
			flags: latest.HelmDeployFlags{
				DepBuild: []string{"--skip-refresh", "--debug"},
			},
			expected: []string{"dep", "build", "chart/path", "--skip-refresh", "--debug"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			h := Helm{
				config: &latest.Helm{
					Flags: test.flags,
				},
			}
			args := h.depBuildArgs(test.chartPath)
			t.CheckDeepEqual(test.expected, args)
		})
	}
}

func TestTemplateArgs(t *testing.T) {
	tests := []struct {
		description    string
		releaseName    string
		release        latest.HelmRelease
		builds         []graph.Artifact
		namespace      string
		additionalArgs []string
		flags          latest.HelmDeployFlags
		expected       []string
	}{
		{
			description: "basic args",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
			},
			expected: []string{"template", "release", "chart/path"},
		},
		{
			description: "with version",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				Version:   "1.2.3",
			},
			expected: []string{"template", "release", "chart/path", "--version", "1.2.3"},
		},
		{
			description: "with namespace",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
			},
			namespace: "namespace",
			expected:  []string{"template", "release", "chart/path", "--namespace", "namespace"},
		},
		{
			description: "with repo",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				Repo:      "repo-url",
			},
			expected: []string{"template", "release", "chart/path", "--repo", "repo-url"},
		},
		{
			description: "with skipTests",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				SkipTests: true,
			},
			expected: []string{"template", "release", "chart/path", "--skip-tests"},
		},
		{
			description:    "with additional args",
			releaseName:    "release",
			release:        latest.HelmRelease{ChartPath: "chart/path"},
			additionalArgs: []string{"--foo", "bar"},
			expected:       []string{"template", "release", "chart/path", "--foo", "bar"},
		},
		{
			description: "with template flags",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
			},
			flags: latest.HelmDeployFlags{
				Template: []string{"--include-crds", "--no-hooks"},
			},
			expected: []string{"template", "release", "chart/path", "--include-crds", "--no-hooks"},
		},
		{
			description: "with set values",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				SetValues: map[string]string{
					"foo": "bar",
					"baz": "qux",
				},
			},
			expected: []string{"template", "release", "chart/path", "--set", "baz=qux", "--set", "foo=bar"},
		},
		{
			description: "with values files",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath:   "chart/path",
				ValuesFiles: []string{"values.yaml", "production-values.yaml"},
			},
			expected: []string{"template", "release", "chart/path", "-f", "values.yaml", "-f", "production-values.yaml"},
		},
		{
			description: "with set values",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				SetValues: map[string]string{
					"app": "test",
				},
			},
			flags: latest.HelmDeployFlags{
				Template: []string{"--include-crds"},
			},
			expected: []string{"template", "release", "chart/path", "--include-crds", "--set", "app=test"},
		},
		{
			description: "with set values overrides and additionalArgs",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				SetValues: map[string]string{
					"app": "test",
				},
			},
			flags: latest.HelmDeployFlags{
				Template: []string{"--include-crds"},
			},
			additionalArgs: []string{"--atomic", "--timeout", "30s"},
			expected:       []string{"template", "release", "chart/path", "--include-crds", "--atomic", "--timeout", "30s", "--set", "app=test"},
		},
		{
			description: "with overrides",
			releaseName: "release",
			release: latest.HelmRelease{
				ChartPath: "chart/path",
				Overrides: util.HelmOverrides{
					Values: map[string]interface{}{
						"app": "test",
						"foo": "bar",
					},
				},
			},
			flags: latest.HelmDeployFlags{
				Template: []string{"--include-crds"},
			},
			expected: []string{"template", "release", "chart/path", "--include-crds", "-f", constants.HelmOverridesFilename},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			h := Helm{
				config: &latest.Helm{
					Flags: test.flags,
				},
			}
			args, err := h.templateArgs(test.releaseName, test.release, test.builds, test.namespace, test.additionalArgs)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, args)
		})
	}
}
