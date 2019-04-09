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

package local

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) buildDocker(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	builder := docker.NewBuilder()
	builder.LocalBuild = b.cfg
	builder.LocalDocker = b.localDocker
	builder.KubeContext = b.runCtx.KubeContext
	builder.PushImages = b.pushImages
	builder.PluginMode = false

	if err := builder.Init(b.runCtx); err != nil {
		return "", nil
	}
	return builder.BuildArtifact(ctx, out, a, tag)
}
