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
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

func (r *SkaffoldRunner) Render(ctx context.Context, out io.Writer, builds []build.Artifact, filepath string) error {
	//Fetch the digest and append it to the tag with the format of "tag@digest"
	if r.runCtx.Opts.SkipBuild {
		for i, a := range builds {
			digest, err := docker.RemoteDigest(a.Tag, r.runCtx.InsecureRegistries)
			if err != nil {
				logrus.Debugf("Digest not found, using %s \n", a.Tag)
				break
			}
			builds[i].Tag = build.TagWithDigest(a.Tag, digest)
		}
	}
	return r.deployer.Render(ctx, out, builds, r.labellers, filepath)
}
