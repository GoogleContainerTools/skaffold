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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// GetRenderer creates a renderer from a given RunContext and pipeline definitions.
func GetRenderer(ctx context.Context, runCtx *runcontext.RunContext, hydrationDir string, labels map[string]string, usingLegacyHelmDeploy bool) (renderer.Renderer, error) {
	configNames := runCtx.Pipelines.AllOrderedConfigNames()

	var gr renderer.GroupRenderer
	var err error
	for _, configName := range configNames {
		p := runCtx.Pipelines.GetForConfigName(configName)
		mkvMap := map[string]string{}
		if runCtx.Opts.ManifestsValueFile != "" {
			mkvMap, err = util.ParseEnvVariablesFromFile(runCtx.Opts.ManifestsValueFile)
			if err != nil {
				return nil, err
			}
		}
		overridesMap := util.EnvSliceToMap(runCtx.Opts.ManifestsOverrides, "=")
		for k := range overridesMap {
			mkvMap[k] = overridesMap[k]
		}

		rs, err := renderer.New(ctx, runCtx, p.Render, hydrationDir, labels, configName, mkvMap)
		if err != nil {
			return nil, err
		}
		gr.Renderers = append(gr.Renderers, rs.Renderers...)
		gr.HookRunners = append(gr.HookRunners, hooks.NewRenderRunner(p.Render.LifecycleHooks, &[]string{runCtx.GetNamespace()},
			hooks.NewRenderEnvOpts(runCtx.KubeContext, []string{runCtx.GetNamespace()}), configName))
	}
	// In case of legacy helm deployer configured and render command used
	// force a helm renderer from deploy helm config
	if usingLegacyHelmDeploy && runCtx.Opts.Command == "render" {
		for _, configName := range configNames {
			p := runCtx.Pipelines.GetForConfigName(configName)
			legacyHelmReleases := filterDuplicates(p.Deploy.LegacyHelmDeploy, p.Render.Helm)
			if len(legacyHelmReleases) == 0 {
				continue
			}
			rCfg := latest.RenderConfig{
				Generate: latest.Generate{
					Helm: &latest.Helm{
						Releases: legacyHelmReleases,
					},
				},
			}
			r, err := helm.New(ctx, runCtx, rCfg, labels, configName, nil)
			if err != nil {
				return nil, err
			}
			gr.Renderers = append(gr.Renderers, r)
		}
	}
	return renderer.NewRenderMux(gr), nil
}

// filterDuplicates removes duplicate releases defined in the legacy helm deployer
func filterDuplicates(l *latest.LegacyHelmDeploy, h *latest.Helm) []latest.HelmRelease {
	if l == nil {
		return nil
	}
	if h == nil {
		return l.Releases
	}
	var rs []latest.HelmRelease
	for i := range l.Releases {
		isDup := false
		for _, r := range h.Releases {
			if r.Name == l.Releases[i].Name {
				isDup = true
				break
			}
		}
		if !isDup {
			rs = append(rs, l.Releases[i])
		}
	}
	return rs
}
