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
	"fmt"
	"io"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// Build builds an artifact with Cloud Native Buildpacks:
// https://buildpacks.io/
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latestV2.Artifact, tag string, matcher platform.Matcher) (string, error) {
	if matcher.IsMultiPlatform() {
		// TODO: Implement building multiplatform images
		log.Entry(ctx).Println("skaffold doesn't yet support multi platform builds for the buildpacks builder")
	}

	built, err := b.build(ctx, out, artifact, tag)
	if err != nil {
		return "", err
	}

	if err := b.localDocker.Tag(ctx, built, tag); err != nil {
		return "", fmt.Errorf("tagging %s->%q: %w", built, tag, err)
	}

	if b.pushImages {
		return b.localDocker.Push(ctx, out, tag)
	}
	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) SupportedPlatforms() platform.Matcher {
	return platform.Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}}
}
