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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// ImageTags maps image names to tags
type ImageTags map[string]string

// Tagger is an interface for tag strategies to be implemented against
type Tagger interface {
	// GenerateTag generates a tag for an artifact.
	GenerateTag(workingDir, imageName string) (string, error)
}

const DeprecatedImageName = "_DEPRECATED_IMAGE_NAME_"

// GenerateFullyQualifiedImageName resolves the fully qualified image name for an artifact.
// The workingDir is the root directory of the artifact with respect to the Skaffold root,
// and imageName is the base name of the image.
func GenerateFullyQualifiedImageName(t Tagger, workingDir, imageName string) (string, error) {
	// Supporting the use of the deprecated {{.IMAGE_NAME}} in envTemplate
	if v, ok := t.(*envTemplateTagger); ok {
		tag, err := v.GenerateTag(workingDir, DeprecatedImageName)

		if err != nil {
			return "", fmt.Errorf("generating envTemplate tag: %w", err)
		}

		if strings.Contains(tag, DeprecatedImageName) {
			warnings.Printf("{{.IMAGE_NAME}} is deprecated, envTemplate's template should only specify the tag value. See https://skaffold.dev/docs/pipeline-stages/taggers/")

			return strings.Replace(tag, DeprecatedImageName, imageName, -1), nil
		}

		return fmt.Sprintf("%s:%s", imageName, tag), nil
	}

	tag, err := t.GenerateTag(workingDir, imageName)
	if err != nil {
		return "", fmt.Errorf("generating tag: %w", err)
	}

	// Do not append :tag to imageName if tag is empty.
	if tag == "" {
		return imageName, nil
	}

	return fmt.Sprintf("%s:%s", imageName, tag), nil
}
