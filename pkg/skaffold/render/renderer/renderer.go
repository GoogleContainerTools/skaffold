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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer/noop"
)

type Renderer interface {
	Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool, output string) error
	// ManifestDeps returns the user kubernetes manifests to file watcher. In dev mode, a "redeploy" will be triggered
	// if any of the "Dependencies" manifest is changed.
	ManifestDeps() ([]string, error)
}

// New creates a new Renderer object from the latestV2 API schema.
func New(cfg render.Config, hydrationDir string, labels map[string]string, usingLegacyHelmDeploy bool) (Renderer, error) {
	renderCfg := cfg.GetRenderConfig()
	if usingLegacyHelmDeploy {
		return noop.New(renderCfg, cfg.GetWorkingDir(), hydrationDir, labels)
	}
	if renderCfg.Validate == nil && renderCfg.Transform == nil && renderCfg.Kpt == nil {
		return kubectl.New(cfg, hydrationDir, labels)
	}

	return kpt.New(cfg, hydrationDir, labels)
}
