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
	Tag            string
	FullyQualified bool
}

// ParseReference parses an image name to a reference.
func ParseReference(image string) (*ImageReference, error) {
	r, err := reference.Parse(image)
	if err != nil {
		return nil, err
	}

	baseName := image
	if n, ok := r.(reference.Named); ok {
		baseName = n.Name()
	}

	fullyQualified := false
	tag := ""
	switch n := r.(type) {
	case reference.Tagged:
		tag = n.Tag()
		fullyQualified = n.Tag() != "latest"
	case reference.Digested:
		fullyQualified = true
	}

	return &ImageReference{
		BaseName:       baseName,
		Tag:            tag,
		FullyQualified: fullyQualified,
	}, nil
}
