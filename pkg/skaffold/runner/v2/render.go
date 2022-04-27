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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

func (r *SkaffoldRunner) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, renderOutputFile string) (manifest.ManifestList, error) {
	if r.runCtx.RenderOnly() {
		// Fetch the digest and append it to the tag with the format of "tag@digest"
		if r.runCtx.DigestSource() == constants.RemoteDigestSource {
			for i, a := range builds {
				digest, err := docker.RemoteDigest(a.Tag, r.runCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve the digest of %s: does the image exist remotely?", a.Tag)
				}
				builds[i].Tag = build.TagWithDigest(a.Tag, digest)
			}
		}
		if r.runCtx.DigestSource() == constants.NoneDigestSource {
			output.Default.Fprintln(out, "--digest-source set to 'none', tags listed in Kubernetes manifests will be used for render")
		}
	}

	ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
	manifestList, errR := r.renderer.Render(ctx, out, builds, offline, renderOutputFile)
	if errR != nil {
		endTrace(instrumentation.TraceEndError(errR))
		return nil, errR
	}
	endTrace()
	return manifestList, nil
}
