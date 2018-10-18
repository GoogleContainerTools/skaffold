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

package local

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// jibBuildImageRef generates a valid image name for the workspace and project.
// The image name is always prefixed with `jib`.
func generateJibImageRef(workspace string, project string) string {
	imageName := "jib" + workspace
	if project != "" {
		imageName += "_" + project
	}
	// if the workspace + project is a valid image name then use it
	if regexp.MustCompile(constants.RepositoryComponentRegex).MatchString(imageName) {
		return imageName
	}
	// otherwise use a hash for a deterministic name
	hasher := sha1.New()
	io.WriteString(hasher, imageName)
	return "jib__" + hex.EncodeToString(hasher.Sum(nil))
}
