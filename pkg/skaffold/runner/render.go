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
	"fmt"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/util"
)

func (r *SkaffoldRunner) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	renderOut, postRenderFn, err := util.WithLogFile(time.Now().Format(util.TimeFormat)+".log", out, r.runCtx.Muted())
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}
	defer postRenderFn()

	eventV2.TaskInProgress(constants.Render, "Render Manifests")
	if r.runCtx.RenderOnly() {
		// Fetch the digest and append it to the tag with the format of "tag@digest"
		if r.runCtx.DigestSource() == constants.RemoteDigestSource {
			for i, a := range builds {
				// remote digest to platform dependant build not supported
				digest, err := docker.RemoteDigest(a.Tag, r.runCtx, nil)
				if err != nil {
					eventV2.TaskFailed(constants.Render, err)
					return manifest.ManifestListByConfig{}, fmt.Errorf("failed to resolve the digest of %s: does the image exist remotely?", a.Tag)
				}
				builds[i].Tag = build.TagWithDigest(a.Tag, digest)
			}
		}
		if r.runCtx.DigestSource() == constants.NoneDigestSource {
			output.Default.Fprintln(out, "--digest-source set to 'none', tags listed in Kubernetes manifests will be used for render")
		}
	}

	ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
	manifestList, err := r.renderer.Render(ctx, renderOut, builds, offline)
	if err != nil {
		eventV2.TaskFailed(constants.Render, err)
		endTrace(instrumentation.TraceEndError(err))
		return manifest.ManifestListByConfig{}, err
	}

	endTrace()
	eventV2.TaskSucceeded(constants.Render)
	return manifestList, nil
}
