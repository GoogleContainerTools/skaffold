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

package docker

import (
	img "github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/image"
)

const maxLength = 255

func SubstituteDefaultRepoIntoImage(defaultRepo string, image string) (string, error) {
	if defaultRepo == "" {
		return image, nil
	}

	parsed, err := ParseReference(image)
	if err != nil {
		return "", err
	}
	defaultRegistry := img.RegistryFactory(defaultRepo)
	originalImage := img.Factory(parsed.BaseName)
	replaced := truncate(originalImage.Update(defaultRegistry))
	if parsed.Tag != "" {
		replaced = replaced + ":" + parsed.Tag
	}
	if parsed.Digest != "" {
		replaced = replaced + "@" + parsed.Digest
	}
	return replaced, nil
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
