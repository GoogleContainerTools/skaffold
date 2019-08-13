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

import "github.com/docker/distribution/reference"

// ImageReference is a parsed image name.
type ImageReference struct {
	BaseName       string
	Domain         string
	Path           string
	Tag            string
	Digest         string
	FullyQualified bool
}

// ParseReference parses an image name to a reference.
func ParseReference(image string) (*ImageReference, error) {
	r, err := reference.Parse(image)
	if err != nil {
		return nil, err
	}

	baseName := image
	var domain string
	var path string
	if n, ok := r.(reference.Named); ok {
		baseName = n.Name()
		domain = reference.Domain(n)
		path = reference.Path(n)
	}

	fullyQualified := false
	tag := ""
	digest := ""

	if n, ok := r.(reference.Tagged); ok {
		tag = n.Tag()
		fullyQualified = n.Tag() != "latest"
	}
	if n, ok := r.(reference.Digested); ok {
		fullyQualified = true
		digest = n.Digest().String()
	}

	return &ImageReference{
		BaseName:       baseName,
		Domain:         domain,
		Path:           path,
		Tag:            tag,
		Digest:         digest,
		FullyQualified: fullyQualified,
	}, nil
}
