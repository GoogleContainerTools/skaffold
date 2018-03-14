/*
Copyright 2018 Google LLC

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
)

// ChecksumTagger tags an image by the sha256 of the image tarball
type ChecksumTagger struct {
	ImageName string
	Checksum  string
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the sha256 checksum of the image
func (c *ChecksumTagger) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("Tag options not provided")
	}
	digestSplit := strings.Split(opts.Digest, ":")
	if len(digestSplit) != 2 {
		return "", fmt.Errorf("Digest wrong format: %s, expected sha256:<checksum>", digestSplit)
	}
	checksum := digestSplit[1]
	return fmt.Sprintf("%s:%s", opts.ImageName, checksum), nil
}
