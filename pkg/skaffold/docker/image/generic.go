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

package image

import (
	"regexp"
	"strings"
)

const maxLength = 255

const gcr = "gcr.io"

var (
	escapeRegex = regexp.MustCompile(`[/._:@]`)
	prefixRegex = regexp.MustCompile(`(.*\.)?gcr.io/[a-zA-Z0-9-_]+/?`)
)

type GenericRegistry struct {
	RegistryName string
}

func NewGenericRegistry(name string) Registry {
	return &GenericRegistry{name}
}

func (r *GenericRegistry) String() string {
	return r.RegistryName
}

func (r *GenericRegistry) Update(reg *Registry) Registry {
	return nil
}

func (r *GenericRegistry) Prefix() string {
	return ""
}

func (r *GenericRegistry) Postfix() string {
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
	defaultRepo := reg.String()
	if defaultRepo == "" {
		return originalImage
	}
	originalPrefix := prefixRegex.FindString(originalImage)
	defaultRepoPrefix := prefixRegex.FindString(defaultRepo)
	if originalPrefix != "" && defaultRepoPrefix != "" {
		// prefixes match
		if originalPrefix == defaultRepoPrefix {
			return defaultRepo + "/" + originalImage[len(originalPrefix):]
		}
		if strings.HasPrefix(originalImage, defaultRepo) {
			return originalImage
		}
		// prefixes don't match, concatenate and truncate
		return defaultRepo + "/" + originalImage
	}

	return reg.String() + "/" + escapeRegex.ReplaceAllString(originalImage, "_")
}
