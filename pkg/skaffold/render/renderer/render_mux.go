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

package renderer

import (
	"context"
	"io"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringset"
)

// RenderMux forwards all method calls to the renderers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns all the manifests.
type RenderMux struct {
	renderers []Renderer
}

func NewRenderMux(renderers []Renderer) Renderer {
	return RenderMux{renderers: renderers}
}

func (r RenderMux) Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool) (manifest.ManifestList, error) {
	allManifests := manifest.ManifestList{}
	for i, renderer := range r.renderers {
		eventV2.RendererInProgress(i)
		w, ctx := output.WithEventContext(ctx, out, constants.Render, strconv.Itoa(i))
		ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
		ms, err := renderer.Render(ctx, w, artifacts, offline)
		if err != nil {
			eventV2.RendererFailed(i, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		allManifests = append(allManifests, ms...)
		eventV2.RendererSucceeded(i)
		endTrace()
	}
	return allManifests, nil
}

func (r RenderMux) ManifestDeps() ([]string, error) {
	deps := stringset.New()
	for _, renderer := range r.renderers {
		result, err := renderer.ManifestDeps()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}
