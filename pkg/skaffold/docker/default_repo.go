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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/registry"
)

const maxLength = 255

var (
	prefixRegex = regexp.MustCompile(`(.*\.)?gcr.io/[a-zA-Z0-9-_]+/?`)
	escapeRegex = regexp.MustCompile(`[/._:@]`)
)

func SubstituteDefaultRepoIntoImage(defaultRepo string, image string, rewriteStrategy bool) (string, error) {
	if rewriteStrategy {
		return substituteDefaultRepoIntoImageNew(defaultRepo, image)
	}
	return substituteDefaultRepoIntoImage(defaultRepo, image)
}

func substituteDefaultRepoIntoImage(defaultRepo string, image string) (string, error) {
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

func substituteDefaultRepoIntoImageNew(defaultRepo string, image string) (string, error) {
	if defaultRepo == "" {
		return image, nil
	}

	parsed, err := ParseReference(image)
	if err != nil {
		return "", err
	}

	// replace registry in parsed name
	var replaced string
	reg, image := splitImage(parsed.BaseName)
	defaultRegistry := registry.GetRegistry(defaultRepo)
	newReg := reg.Update(defaultRegistry)
	if newReg.Type() == reg.Type() {
		replaced = newReg.Name() + "/" + image
	} else {
		replaced = newReg.Name() + "/" + escapeRegex.ReplaceAllString(parsed.BaseName, "_")
	}
	replaced = truncate(replaced)

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

func splitImage(i string) (registry.Registry, string) {
	s := strings.Split(i, "/")
	reg := registry.GetRegistry(strings.Join(s[:len(s)-1], "/"))
	return reg, s[len(s)-1]
}
