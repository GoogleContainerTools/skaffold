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

package buildpacks

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// BuildpackBuilder is a builder for buildpack artifacts
type BuildpackBuilder struct {
	localDocker docker.LocalDaemon
	pushImages  bool
}

// NewArtifactBuilder returns a new buildpack artifact builder
func NewArtifactBuilder(localDocker docker.LocalDaemon, pushImages bool) *BuildpackBuilder {
	return &BuildpackBuilder{
		localDocker: localDocker,
		pushImages:  pushImages,
	}
}

// Build builds an artifact with Cloud Native Buildpacks:
// https://buildpacks.io/
func (b *BuildpackBuilder) Build(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	built, err := b.build(ctx, out, a.Workspace, a.BuildpackArtifact, tag)
	if err != nil {
		return "", err
	}

	if err := b.localDocker.Tag(ctx, built, tag); err != nil {
		return "", errors.Wrapf(err, "tagging %s->%s", built, tag)
	}

	if b.pushImages {
		return b.localDocker.Push(ctx, out, tag)
	}
	return b.localDocker.ImageID(ctx, tag)
}
