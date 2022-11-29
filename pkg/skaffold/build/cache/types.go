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

package cache

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
)

type BuildAndTestFn func(context.Context, io.Writer, tag.ImageTags, []*latest.Artifact, platform.Resolver) ([]graph.Artifact, error)

type Cache interface {
	Build(context.Context, io.Writer, tag.ImageTags, []*latest.Artifact, platform.Resolver, BuildAndTestFn) ([]graph.Artifact, error)
	AddArtifact(ctx context.Context, a graph.Artifact) error
}

type noCache struct{}

func (n *noCache) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, platforms platform.Resolver, buildAndTest BuildAndTestFn) ([]graph.Artifact, error) {
	return buildAndTest(ctx, out, tags, artifacts, platforms)
}

func (n *noCache) AddArtifact(ctx context.Context, a graph.Artifact) error {
	// noop
	return nil
}
