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
func (c *ChecksumTagger) GenerateFullyQualifiedImageName() (string, error) {
	return fmt.Sprintf("%s:%s", c.ImageName, c.Checksum), nil
}

// NewChecksumTaggerFromDigest returns a tagger instance by splitting the docker digest
// into identifier and checksum
func NewChecksumTaggerFromDigest(digest, imageName string) (*ChecksumTagger, error) {
	digestSplit := strings.Split(digest, ":")
	if len(digestSplit) != 2 {
		return nil, fmt.Errorf("Digest wrong format: %s, expected sha256:<checksum>", digestSplit)
	}
	checksum := digestSplit[1]
	return &ChecksumTagger{
		ImageName: imageName,
		Checksum:  checksum,
	}, nil
}
