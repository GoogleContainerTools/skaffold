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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/registry"
)

const maxLength = 255

func SubstituteDefaultRepoIntoImage(defaultRepo string, image string, rewriteStrategy bool) (string, error) {
	if defaultRepo == "" {
		return image, nil
	}
	parsed, err := ParseReference(image)
	if err != nil {
		return "", err
	}

	var replaced string
	if rewriteStrategy {
		replaced = substituteDefaultRepoIntoImageWithRewrite(defaultRepo, parsed.BaseName)
	} else {
		replaced = replace(defaultRepo, parsed.BaseName)
	}
	if parsed.Tag != "" {
		replaced = replaced + ":" + parsed.Tag
	}
	if parsed.Digest != "" {
		replaced = replaced + "@" + parsed.Digest
	}
	return truncate(replaced), nil
}

func replace(defaultRepo string, baseImage string) string {
	originalPrefix := registry.GCRPrefixRegex.FindString(baseImage)
	defaultRepoPrefix := registry.GCRPrefixRegex.FindString(defaultRepo)
	if originalPrefix != "" && defaultRepoPrefix != "" {
		// prefixes match
		if originalPrefix == defaultRepoPrefix {
			return defaultRepo + "/" + baseImage[len(originalPrefix):]
		}
		if strings.HasPrefix(baseImage, defaultRepo) {
			return baseImage
		}
		// prefixes don't match, concatenate image string
		return defaultRepo + "/" + baseImage
	}
	return defaultRepo + "/" + registry.ESCRegex.ReplaceAllString(baseImage, registry.ReplaceStr)
}

func truncate(image string) string {
	if len(image) > maxLength {
		return image[0:maxLength]
	}
	return image
}

func substituteDefaultRepoIntoImageWithRewrite(defaultRepo string, image string) string {	
	oldRepo, image := splitImage(image)
	defaultRepository := registry.New(defaultRepo)
	newRepo := oldRepo.Update(defaultRepository)
	// if both repository types are similar, then image will be re-written
	if newRepo.Type() == oldRepo.Type() {
		return newRepo.Name() + "/" + image
	}
	// If repository are of different type, then rewrite image as new repository name,
	// prefix from old image and then finally image name.
	// e.g. if image is docker hub image "mcr.microsoft.com/windows/servercore/image" and
	// default repo is gcr.io/my-project
	// then Generic Repository is "mcr.microsoft.com/windows/servercore/" and
	// image name is "image"
	// the prefix for Generic Repository  "mcr.microsoft.com/windows/servercore" is "mcr.microsoft.com_windows_servercore"
	// the final replaced image would be "gcr.io/my-project/mcr.microsoft.com_windows_servercore_image
	if oldRepo.Prefix() == "" {
		return newRepo.Name() + "/" + image
	}
	return newRepo.Name() + "/" + strings.Join([]string{oldRepo.Prefix(), image}, "_")
}

func splitImage(i string) (registry.Registry, string) {
	s := strings.Split(i, "/")
	reg := registry.New(strings.Join(s[:len(s)-1], "/"))
	return reg, s[len(s)-1]
}
