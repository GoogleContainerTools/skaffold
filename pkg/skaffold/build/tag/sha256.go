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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// ChecksumTagger tags an image by the sha256 of the image tarball
type ChecksumTagger struct{}

// Labels are labels specific to the sha256 tagger.
func (c *ChecksumTagger) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "sha256",
	}
}

func (c *ChecksumTagger) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	parsed, err := docker.ParseReference(imageName)
	if err != nil {
		return "", err
	}

	if parsed.Tag == "" {
		// No supplied tag, so use "latest".
		return imageName + ":latest", nil
	}

	// They already have a tag.
	return imageName, nil
}
