/*
Copyright 2018 The Skaffold Authors

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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/pkg/errors"
)

// ChecksumTagger tags an image by the sha256 of the image tarball
type ChecksumTagger struct{}

// Labels are labels specific to the sha256 tagger.
func (c *ChecksumTagger) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "sha256",
	}
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the sha256 checksum of the image
func (c *ChecksumTagger) GenerateFullyQualifiedImageName(workingDir string, opts *Options) (string, error) {
	if opts == nil {
		return "", errors.New("tag options not provided")
	}

	digest := opts.Digest
	sha256 := strings.TrimPrefix(opts.Digest, "sha256:")
	if sha256 == digest {
		return "", fmt.Errorf("digest wrong format: %s, expected sha256:<checksum>", digest)
	}

	return fmt.Sprintf("%s:%s", opts.ImageName, sha256), nil
}
