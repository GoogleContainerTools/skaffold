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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// ChecksumTagger tags an image by the sha256 of the image tarball
type ChecksumTagger struct{}

// Labels are labels specific to the sha256 tagger.
func (t *ChecksumTagger) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "sha256",
	}
}

// GenerateTag resolves the tag portion of the fully qualified image name for an artifact.
func (t *ChecksumTagger) GenerateTag(workingDir, imageName string) (string, error) {
	parsed, err := docker.ParseReference(imageName)
	if err != nil {
		return "", err
	}

	if parsed.Tag == "" {
		// No supplied tag, so use "latest".
		return ":latest", nil
	}

	//They already have a tag.
	return "", nil
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the git commit.
func (t *ChecksumTagger) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	tag, err := t.GenerateTag(workingDir, imageName)
	if err != nil {
		return "", fmt.Errorf("generating tag: %w", err)
	}
	return imageName + tag, nil
}
