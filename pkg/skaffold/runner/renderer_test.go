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

package runner

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetRenderer(tOuter *testing.T) {
	rc := &runcontext.RunContext{
		Pipelines: runcontext.NewPipelines(
			map[string]latest.Pipeline{
				"default": {},
			}, []string{"default"})}
	labels := map[string]string{}
	kubectlCfg := latest.RenderConfig{
		Generate: latest.Generate{
			RawK8s: []string{"k8s/*"}},
	}
	helmConfig := latest.RenderConfig{
		Generate: latest.Generate{
			Helm: &latest.Helm{
				Releases: []latest.HelmRelease{
					{Name: "test", ChartPath: "./test"},
					{Name: "test1", ChartPath: "./test1"},
				},
			},
		},
	}
	kptConfig := latest.RenderConfig{
		Generate: latest.Generate{
			Kpt: []string{"kptfile"},
		},
	}
	testutil.Run(tOuter, "TestGetRenderer", func(t *testutil.T) {
		tests := []struct {
			description       string
			cfg               latest.Pipeline
			expected          renderer.Renderer
			apply             bool
			shouldErr         bool
			deepCheckRenderer bool
		}{
			{
				description: "no renderer",
				expected:    renderer.RenderMux{},
			},
			{
				description: "legacy helm deployer",
				cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							LegacyHelmDeploy: &latest.LegacyHelmDeploy{
								Releases: []latest.HelmRelease{
									{Name: "test", ChartPath: "./test"},
									{Name: "test1", ChartPath: "./test1"},
								},
							},
						},
					},
				},
				// expected: renderer.NewRenderMux([]renderer.Renderer{
				// 	t.RequireNonNilResult(helm.New(rc, helmConfig, labels, "")).(renderer.Renderer)}),
				expected: renderer.NewRenderMux(
					renderer.GroupRenderer{
						Renderers: []renderer.Renderer{
							t.RequireNonNilResult(helm.New(rc, helmConfig, labels, "", nil)).(renderer.Renderer)},
					},
				),
			},
			{
				description: "helm renderer",
				cfg: latest.Pipeline{
					Render: helmConfig,
				},
				expected: renderer.NewRenderMux(
					renderer.GroupRenderer{
						Renderers: []renderer.Renderer{
							t.RequireNonNilResult(helm.New(rc, helmConfig, labels, "", nil)).(renderer.Renderer)},
					},
				),
			},
			{
				description: "kubectl renderer",
				cfg: latest.Pipeline{
					Render: kubectlCfg,
				},
				expected: renderer.NewRenderMux(
					renderer.GroupRenderer{
						Renderers: []renderer.Renderer{
							t.RequireNonNilResult(helm.New(rc, kubectlCfg, labels, "", nil)).(renderer.Renderer)},
					},
				),
			},
			{
				description: "kpt renderer",
				cfg: latest.Pipeline{
					Render: kptConfig,
				},
				expected: renderer.NewRenderMux(
					renderer.GroupRenderer{
						Renderers: []renderer.Renderer{
							t.RequireNonNilResult(helm.New(rc, kptConfig, labels, "", nil)).(renderer.Renderer)},
					},
				),
			},
			{
				description: "kpt renderer when validate configured",
				cfg: latest.Pipeline{
					Render: latest.RenderConfig{
						Generate: latest.Generate{RawK8s: []string{"test"}},
						Validate: &[]latest.Validator{{Name: "kubeval"}},
					},
				},
				expected: renderer.NewRenderMux(
					renderer.GroupRenderer{
						Renderers: []renderer.Renderer{
							t.RequireNonNilResult(helm.New(rc, kptConfig, labels, "", nil)).(renderer.Renderer)},
					},
				),
			},
		}
		for _, test := range tests {
			testutil.Run(tOuter, test.description, func(t *testutil.T) {
				rs, err := GetRenderer(context.Background(), &runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(
						map[string]latest.Pipeline{
							"default": test.cfg,
						},
						[]string{"default"}),
				}, "", map[string]string{}, false)

				t.CheckError(test.shouldErr, err)
				t.CheckTypeEquality(test.expected, rs)
			})
		}
	})
}
