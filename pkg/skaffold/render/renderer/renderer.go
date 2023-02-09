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

package renderer

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kpt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kustomize"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Renderer interface {
	Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error)
	// ManifestDeps returns the user kubernetes manifests to file watcher. In dev mode, a "redeploy" will be triggered
	// if any of the "Dependencies" manifest is changed.
	ManifestDeps() ([]string, error)
}

// New creates a new Renderer object from the latestV2 API schema.
func New(ctx context.Context, cfg render.Config, renderCfg latest.RenderConfig, hydrationDir string, labels map[string]string, configName string, manifestOverrides map[string]string) (GroupRenderer, error) {
	var rs GroupRenderer
	rs.HookRunners = []hooks.Runner{hooks.NewRenderRunner(renderCfg.Generate.LifecycleHooks, &[]string{cfg.GetNamespace()}, hooks.NewRenderEnvOpts(cfg.GetKubeContext(), []string{cfg.GetNamespace()}))}

	if renderCfg.Kpt != nil {
		r, err := kpt.New(cfg, renderCfg, hydrationDir, labels, configName, cfg.GetNamespace(), manifestOverrides)
		if err != nil {
			return GroupRenderer{}, err
		}
		log.Entry(ctx).Infof("setting up kpt renderer")
		rs.Renderers = append(rs.Renderers, r)
	}

	if renderCfg.RawK8s != nil || renderCfg.RemoteManifests != nil {
		r, err := kubectl.New(cfg, renderCfg, labels, configName, cfg.GetNamespace(), manifestOverrides)
		if err != nil {
			return GroupRenderer{}, err
		}
		rs.Renderers = append(rs.Renderers, r)
	}
	if renderCfg.Kustomize != nil {
		r, err := kustomize.New(cfg, renderCfg, labels, configName, cfg.GetNamespace(), manifestOverrides)
		if err != nil {
			return GroupRenderer{}, err
		}
		rs.Renderers = append(rs.Renderers, r)
	}

	if renderCfg.Helm != nil {
		r, err := helm.New(cfg, renderCfg, labels, configName, manifestOverrides)
		if err != nil {
			return GroupRenderer{}, err
		}
		rs.Renderers = append(rs.Renderers, r)
	}
	return rs, nil
}
