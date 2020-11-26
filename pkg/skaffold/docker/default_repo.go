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

	"github.com/docker/distribution/reference"
)

const maxLength = 255

var (
	escapeRegex = regexp.MustCompile(`[/._:@]`)
	// gcpProjectIDRegex matches a GCP Project ID as according to console.cloud.google.com.
	gcpProjectIDRegex = `[a-z][a-z0-9-]{4,28}[a-z0-9]`
	// prefixRegex is used to match a GCR or AR reference, which must have a project ID.
	prefixRegex = regexp.MustCompile(`^` + reference.DomainRegexp.String() + `/` + gcpProjectIDRegex + `/?`)
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
	if strings.HasPrefix(baseImage, defaultRepo) {
		return baseImage
	}
	originalPrefix := prefixRegex.FindString(baseImage)
	defaultRepoPrefix := prefixRegex.FindString(defaultRepo)
	if registrySupportsMultiLevelRepos(defaultRepoPrefix) {
		// prefixes match
		if originalPrefix == defaultRepoPrefix {
			return defaultRepo + "/" + baseImage[len(originalPrefix):]
		}
		// prefixes don't match, concatenate and truncate
		return truncate(defaultRepo + "/" + baseImage)
	}

	return truncate(defaultRepo + "/" + escapeRegex.ReplaceAllString(baseImage, "_"))
}

func registrySupportsMultiLevelRepos(repo string) bool {
	return strings.Contains(repo, "gcr.io") || strings.Contains(repo, "-docker.pkg.dev")
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
