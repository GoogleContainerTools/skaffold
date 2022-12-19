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

package tag

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// ChecksumTagger tags an image by the sha256 of the image tarball
type ChecksumTagger struct{}

// GenerateTag returns either the current tag or `latest`.
func (t *ChecksumTagger) GenerateTag(ctx context.Context, image latest.Artifact) (string, error) {
	parsed, err := docker.ParseReference(image.ImageName)
	if err != nil {
		return "", err
	}

	if parsed.Tag == "" {
		// No supplied tag, so use "latest".
		return "latest", nil
	}

	// imageName already has a tag
	return "", nil
}
