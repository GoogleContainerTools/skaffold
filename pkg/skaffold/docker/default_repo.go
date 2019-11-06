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
	"regexp"
	"strings"
)

const maxLength = 255

var (
	escapeRegex = regexp.MustCompile(`[/._:@]`)
	prefixRegex = regexp.MustCompile(`(.*\.)?gcr.io/[a-zA-Z0-9-_]+/?`)
)

func SubstituteDefaultRepoIntoImage(defaultRepo string, image string) (string, error) {
	if defaultRepo == "" {
		return image, nil
	}

	parsed, err := ParseReference(image)
	if err != nil {
		return "", err
	}

	replaced := replace(defaultRepo, parsed.BaseName)
	if parsed.Tag != "" {
		replaced = replaced + ":" + parsed.Tag
	}
	if parsed.Digest != "" {
		replaced = replaced + "@" + parsed.Digest
	}

	return replaced, nil
}

func replace(defaultRepo string, baseImage string) string {
	originalPrefix := prefixRegex.FindString(baseImage)
	defaultRepoPrefix := prefixRegex.FindString(defaultRepo)
	if originalPrefix != "" && defaultRepoPrefix != "" {
		// prefixes match
		if originalPrefix == defaultRepoPrefix {
			return defaultRepo + "/" + baseImage[len(originalPrefix):]
		}
		if strings.HasPrefix(baseImage, defaultRepo) {
			return baseImage
		}
		// prefixes don't match, concatenate and truncate
		return truncate(defaultRepo + "/" + baseImage)
	}

	return truncate(defaultRepo + "/" + escapeRegex.ReplaceAllString(baseImage, "_"))
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
