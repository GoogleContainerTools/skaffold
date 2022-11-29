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
	"context"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
)

func StripTags(taggedImages []string, ignoreDigest bool) []string {
	// Remove tags from image names
	var images []string
	for _, image := range taggedImages {
		tag := StripTag(image, ignoreDigest)
		if tag != "" {
			images = append(images, tag)
		}
	}
	return images
}

func StripTag(image string, ignoreDigest bool) string {
	parsed, err := docker.ParseReference(image)
	if err != nil {
		// It's possible that it's a templatized name that can't be parsed as is.
		olog.Entry(context.TODO()).Debugf("Couldn't parse image [%s]: %s", image, err.Error())
		return ""
	}
	if ignoreDigest && parsed.Digest != "" {
		warnings.Printf("Ignoring image referenced by digest: [%s]", image)
		return ""
	}

	return parsed.BaseName
}

func SetImageTag(image, tag string) (string, error) {
	parsed, err := docker.ParseReference(image)
	if err != nil {
		return "", err
	}
	image = parsed.BaseName
	if tag != "" {
		image = image + ":" + tag
	}
	if parsed.Digest != "" {
		image = image + "@" + parsed.Digest
	}
	return image, nil
}
