/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"context"
	"fmt"
	"io"
	"strings"

	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

func SupportsMultiPlatformBuild(a latest.Artifact) bool {
	switch {
	case a.DockerArtifact != nil || a.BazelArtifact != nil || a.BuildpackArtifact != nil:
		return false
	case a.JibArtifact != nil || a.CustomArtifact != nil || a.KoArtifact != nil:
		return true
	default:
		return false
	}
}

func CreateMultiPlatformImage(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher, ab ArtifactBuilder) (string, error) {
	images, err := buildImageForPlatforms(ctx, out, a, tag, matcher, ab)
	if err != nil {
		return "", err
	}

	return docker.CreateManifestList(ctx, images, tag)
}

func buildImageForPlatforms(ctx context.Context, out io.Writer, a *latest.Artifact, tag string, matcher platform.Matcher, ab ArtifactBuilder) ([]docker.SinglePlatformImage, error) {
	var images []docker.SinglePlatformImage

	for _, p := range matcher.Platforms {
		m := platform.Matcher{
			All:       false,
			Platforms: []specs.Platform{p},
		}
		tagWithPlatform := fmt.Sprintf("%s_%s", tag, strings.ReplaceAll(platform.Format(p), "/", "_"))
		imageID, err := ab(ctx, out, a, tagWithPlatform, m)

		if err != nil {
			return nil, err
		}

		pl := util.ConvertToV1Platform(p)
		images = append(images, docker.SinglePlatformImage{
			Platform: &pl,
			Image:    imageID,
		})
	}

	return images, nil
}
