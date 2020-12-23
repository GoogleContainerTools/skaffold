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

var (
	// https://github.com/google/ko#the-ko-model
	allowedImagePrefixes = []string{`ko://`}

	// https://github.com/google/re2/wiki/Syntax
	caseInsensitiveReferenceRegexp = regexp.MustCompile(`(?i)` + reference.ReferenceRegexp.String())
)

// ImageReference is a parsed image name.
type ImageReference struct {
	BaseName       string
	Domain         string
	Path           string
	Repo           string
	Name           string
	Tag            string
	Digest         string
	FullyQualified bool
}

// ParseReference parses an image name to a reference.
func ParseReference(image string) (*ImageReference, error) {
	image = SanitizeImageName(image)
	r, err := reference.Parse(image)
	if err != nil {
		return nil, err
	}

	parsed := &ImageReference{
		BaseName: image,
	}

	if n, ok := r.(reference.Named); ok {
		parsed.BaseName = n.Name()
		parsed.Domain = reference.Domain(n)
		parsed.Path = reference.Path(n)
	}

	if n, ok := r.(reference.Tagged); ok {
		parsed.Tag = n.Tag()
		parsed.FullyQualified = n.Tag() != "latest"
	}

	if n, ok := r.(reference.Digested); ok {
		parsed.Digest = n.Digest().String()
		parsed.FullyQualified = true
	}

	repoParts := strings.Split(parsed.BaseName, "/")
	if len(repoParts) > 1 {
		parsed.Repo = strings.Join(repoParts[:len(repoParts)-1], "/")
		parsed.Name = repoParts[len(repoParts)-1]
	}

	return parsed, nil
}

// SanitizeImageName removes prefixes and lowercases the name portion of images.
// This is primarily used to handle `ko` import paths in image fields, e.g.
// `ko://github.com/GoogleContainerTools/skaffold`.
// Tags can contain uppercase characters, so this function takes care not to
// change tag cases.
func SanitizeImageName(image string) string {
	for _, prefix := range allowedImagePrefixes {
		image = strings.TrimPrefix(image, prefix)
	}
	matches := caseInsensitiveReferenceRegexp.FindStringSubmatch(image)
	if len(matches) > 1 {
		image = strings.Replace(image, matches[1], strings.ToLower(matches[1]), 1)
	}
	return image
}
