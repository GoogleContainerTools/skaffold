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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
)

// GetRenderer creates a renderer from a given RunContext and pipeline definitions.
func GetRenderer(ctx context.Context, runCtx *runcontext.RunContext, hydrationDir string, labels map[string]string, usingLegacyHelmDeploy bool) (renderer.Renderer, error) {
	rs := runCtx.Renderers()

	var renderers []renderer.Renderer
	for _, r := range rs {
		r, err := renderer.New(runCtx, r, hydrationDir, labels, usingLegacyHelmDeploy)
		if err != nil {
			return nil, err
		}
		renderers = append(renderers, r)
	}
	return renderer.NewRenderMux(renderers), nil
}
