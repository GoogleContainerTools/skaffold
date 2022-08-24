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

package runner

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// GetRenderer creates a renderer from a given RunContext and pipeline definitions.
func GetRenderer(ctx context.Context, runCtx *runcontext.RunContext, hydrationDir string, labels map[string]string, usingLegacyHelmDeploy bool) (renderer.Renderer, error) {
	ps := runCtx.Pipelines.AllByConfigNames()

	var gr renderer.GroupRenderer
	gr.HookRunner = hooks.NewRenderRunner(runCtx.GetRenderConfig().LifecycleHooks, &[]string{runCtx.GetNamespace()},
		hooks.NewRenderEnvOpts(runCtx.KubeContext, []string{runCtx.GetNamespace()}))
	for configName, p := range ps {
		rs, err := renderer.New(ctx, runCtx, p.Render, hydrationDir, labels, configName)
		if err != nil {
			return nil, err
		}
		gr.Renderers = append(gr.Renderers, rs.Renderers...)
		gr.HookRunner = rs.HookRunner
	}
	// In case of legacy helm deployer configured and render command used
	// force a helm renderer from deploy helm config
	if usingLegacyHelmDeploy && runCtx.Opts.Command == "render" {
		for configName, p := range ps {
			if p.Deploy.LegacyHelmDeploy == nil {
				continue
			}
			rCfg := latest.RenderConfig{
				Generate: latest.Generate{
					Helm: &latest.Helm{
						Releases: p.Deploy.LegacyHelmDeploy.Releases,
					},
				},
			}
			r, err := helm.New(runCtx, rCfg, labels, configName)
			if err != nil {
				return nil, err
			}
			gr.Renderers = append(gr.Renderers, r)
		}
	}
	return renderer.NewRenderMux(gr), nil
}
