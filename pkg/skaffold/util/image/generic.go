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

package image

import (
	"strings"
)

const maxLength = 255

const gcr = "gcr.io"

//const escapeChars = "[/._:@]"
//const prefixRegexStr = "gcr.io/[a-zA-Z0-9-_]+/"

type GenericContainerRegistry struct {
	RegistryName string
}

func NewGenericContainerRegistry(name string) Registry {
	return &GenericContainerRegistry{name}
}

func (r *GenericContainerRegistry) String() string {
	return r.RegistryName
}

func (r *GenericContainerRegistry) Update(reg *Registry) Registry {
	return nil
}

func (r *GenericContainerRegistry) Prefix() string {
	return ""
}

func (r *GenericContainerRegistry) Postfix() string {
	return ""
}

type GenericImage struct {
	ImageRegistry Registry
	ImageName     string
}

func NewGenericImage(reg Registry, name string) *GenericImage {
	return &GenericImage{reg, name}
}

func (i *GenericImage) Registry() Registry {
	return i.ImageRegistry
}

func (i *GenericImage) String() string {
	return i.ImageName
}

func (i *GenericImage) Update(reg Registry) string {
	// In the case that defaultRepo is an emptystring, we don't want a slash at the start
	originalImage := i.String()
	if len(i.ImageRegistry.String()) != 0 {
		originalImage = i.ImageRegistry.String() + "/" + originalImage
	}

	if reg.String() == "" {
		return originalImage
	}
	if strings.HasPrefix(reg.String(), gcr) {
		originalPrefix := prefixRegex.FindString(originalImage)
		defaultRepoPrefix := prefixRegex.FindString(reg.String())

		if originalPrefix == defaultRepoPrefix {
			// prefixes match
			return reg.String() + "/" + originalImage[len(originalPrefix):]
		} else if strings.HasPrefix(originalImage, reg.String()) {
			return originalImage
		}
		// prefixes don't match, concatenate and truncate
		return truncate(reg.String() + "/" + originalImage)
	}
	return truncate(reg.String() + "/" + escapeRegex.ReplaceAllString(originalImage, "_"))
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}
