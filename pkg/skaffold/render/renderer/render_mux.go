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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
)

// GroupRenderer maintains the slice of all `Renderer`s and their respective lifecycle hooks defined in a single Skaffold config.
type GroupRenderer struct {
	Renderers   []Renderer
	HookRunners []hooks.RenderHookRunner
}

// RenderMux forwards all method calls to the renderers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns all the manifests.
type RenderMux struct {
	gr GroupRenderer
}

func NewRenderMux(renderers GroupRenderer) Renderer {
	return RenderMux{gr: renderers}
}

func (r RenderMux) Render(ctx context.Context, out io.Writer, artifacts []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	allManifests := manifest.NewManifestListByConfig()

	w, ctx := output.WithEventContext(ctx, out, constants.Render, constants.SubtaskIDNone)
	for i := range r.gr.HookRunners {
		if err := r.gr.HookRunners[i].RunPreHooks(ctx, w); err != nil {
			return manifest.ManifestListByConfig{}, err
		}
	}

	for i, renderer := range r.gr.Renderers {
		eventV2.RendererInProgress(i)
		w, ctx = output.WithEventContext(ctx, out, constants.Render, strconv.Itoa(i))
		ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
		manifestsByConfig, err := renderer.Render(ctx, w, artifacts, offline)
		if err != nil {
			eventV2.RendererFailed(i, err)
			endTrace(instrumentation.TraceEndError(err))
			return manifest.NewManifestListByConfig(), err
		}

		for _, configName := range manifestsByConfig.ConfigNames() {
			manifests := manifestsByConfig.GetForConfig(configName)

			if len(manifests) > 0 {
				allManifests.Add(configName, manifests)
			}
		}
		eventV2.RendererSucceeded(i)
		endTrace()
	}
	w, ctx = output.WithEventContext(ctx, out, constants.Render, constants.SubtaskIDNone)

	if len(r.gr.HookRunners) == 0 {
		return allManifests, nil
	}

	updated := manifest.NewManifestListByConfig()
	for _, name := range allManifests.ConfigNames() {
		list := allManifests.GetForConfig(name)
		found := false
		for _, hr := range r.gr.HookRunners {
			if hr.GetConfigName() == name {
				found = true
				if l, err := hr.RunPostHooks(ctx, list, w); err != nil {
					return manifest.ManifestListByConfig{}, err
				} else {
					updated.Add(hr.GetConfigName(), l)
				}
			}
		}
		if !found {
			updated.Add(name, list)
		}
	}
	return updated, nil
}

func (r RenderMux) ManifestDeps() ([]string, error) {
	deps := stringset.New()
	for _, renderer := range r.gr.Renderers {
		result, err := renderer.ManifestDeps()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}
