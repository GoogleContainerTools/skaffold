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

package build

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

// MergeWithPreviousBuilds merges previous or prebuilt build artifacts with
// builds. If an artifact is already present in builds, the same artifact from
// previous will be replaced at the same position.
func MergeWithPreviousBuilds(builds, previous []graph.Artifact) []graph.Artifact {
	updatedBuilds := map[string]graph.Artifact{}
	for _, build := range builds {
		updatedBuilds[build.ImageName] = build
	}

	added := map[string]bool{}
	var merged []graph.Artifact

	for _, artifact := range previous {
		if updated, found := updatedBuilds[artifact.ImageName]; found {
			merged = append(merged, updated)
		} else {
			merged = append(merged, artifact)
		}
		added[artifact.ImageName] = true
	}

	for _, artifact := range builds {
		if !added[artifact.ImageName] {
			merged = append(merged, artifact)
		}
	}

	return merged
}

func TagWithDigest(tag, digest string) string {
	digestSuffix := "@" + digest
	if strings.HasSuffix(tag, digestSuffix) {
		return tag
	}
	return tag + digestSuffix
}

func TagWithImageID(ctx context.Context, tag string, imageID string, localDocker docker.LocalDaemon) (string, error) {
	return localDocker.TagWithImageID(ctx, tag, imageID)
}
