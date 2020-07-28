/*
Copyright 2019 The Skaffold Authors

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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

func (r *SkaffoldRunner) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	//Fetch the digest and append it to the tag with the format of "tag@digest"
	if r.runCtx.Opts.DigestSource == remoteDigestSource {
		for i, a := range builds {
			digest, err := docker.RemoteDigest(a.Tag, r.runCtx.InsecureRegistries)
			if err != nil {
				return fmt.Errorf("failed to resolve the digest of %s, render aborted", a.Tag)
			}
			builds[i].Tag = build.TagWithDigest(a.Tag, digest)
		}
	}
	if r.runCtx.Opts.DigestSource == noneDigestSource {
		color.Default.Fprintln(out, "--digest-source set to 'none', tags listed in Kubernetes manifests will be used for render")
	}
	return r.deployer.Render(ctx, out, builds, offline, filepath)
}
