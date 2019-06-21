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

package util

import (
	"regexp"
	"strings"
)

const maxLength = 255

const gcr = "gcr.io"
const escapeChars = "[/._:@]"
const prefixRegexStr = "gcr.io/[a-zA-Z0-9-_]+/"

var escapeRegex = regexp.MustCompile(escapeChars)
var prefixRegex = regexp.MustCompile(prefixRegexStr)

type Registry interface {
	// Name returns the string representation of the registry
	String() string

	// Replace replaces the current registry in a given image name to input registry
	Update(reg *Registry) Registry

	// Prefix gives the prefix for replacing the registry
	Prefix() string

	// Postfix gives the postfix for replacing the registry
	Postfix() string
}

type Image interface {
	// Registry returns the registry for a given image
	Registry() *Registry

	// Name returns the image name
	String() string

	// Replace updates the Registry for the image to a new Registry and returns the updated Image
	Update(reg *Registry) string
}

func SubstituteDefaultRepoIntoImage(defaultRepo string, originalImage string) string {
	if defaultRepo == "" {
		return originalImage
	}
	if strings.HasPrefix(defaultRepo, gcr) {
		originalPrefix := prefixRegex.FindString(originalImage)
		defaultRepoPrefix := prefixRegex.FindString(defaultRepo)

		if originalPrefix == defaultRepoPrefix {
			// prefixes match
			return defaultRepo + "/" + originalImage[len(originalPrefix):]
		} else if strings.HasPrefix(originalImage, defaultRepo) {
			return originalImage
		}
		// prefixes don't match, concatenate and truncate
		return truncate(defaultRepo + "/" + originalImage)
	}
	return truncate(defaultRepo + "/" + escapeRegex.ReplaceAllString(originalImage, "_"))
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
