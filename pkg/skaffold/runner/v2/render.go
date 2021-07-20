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
package v2

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

func (r *SkaffoldRunner) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
	if err := r.renderer.Render(ctx, out, builds, offline, filepath); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func getRenderConfig(runCtx *runcontext.RunContext) *latestV2.RenderConfig {
	p := runCtx.GetPipelines()
	if len(p) > 0 {
		// TODO: support multi-config
		return &p[0].Render
	}
	return &latestV2.RenderConfig{}
}
