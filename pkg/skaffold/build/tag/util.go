/*
Copyright 2020 The Skaffold Authors

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

func StripTags(taggedImages []string) []string {
	// Remove tags from image names
	var images []string
	for _, image := range taggedImages {
		parsed, err := docker.ParseReference(image)
		if err != nil {
			// It's possible that it's a templatized name that can't be parsed as is.
			warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
			continue
		}
		if parsed.Digest != "" {
			warnings.Printf("Ignoring image referenced by digest: [%s]", image)
			continue
		}

		images = append(images, parsed.BaseName)
	}
	return images
}
