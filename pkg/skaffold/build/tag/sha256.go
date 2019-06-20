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
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
		checksum, err := generateCheckSum(workingDir)
		//If impossible to generate a checksum for the working dir
		//we set the tag to latest by default.
		if err != nil {
			return imageName + ":latest", nil
		}

		return imageName + ":" + checksum, nil
	}

	// They already have a tag.
	return imageName, nil
}

func generateCheckSum(workingDir string) (string, error) {

	var checksum []byte

	err := filepath.Walk(workingDir,
		func(path string, info os.FileInfo, err error) error {
			//if Walk failed on a given file, we skip it.
			// The checksum will be compute on all the other files.
			if err != nil {
				return nil
			}

			f, err := os.Open(path)
			//if file cannot be opened, we skip it. See above.
			if err != nil {
				return nil
			}
			defer f.Close()

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return nil
			}
			checksum = h.Sum(checksum)
			return nil
		})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5.Sum(checksum)), nil
}
