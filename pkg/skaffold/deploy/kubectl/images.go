/*
Copyright 2018 The Skaffold Authors

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

package kubectl

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// for testing
var warner Warner = &logrusWarner{}

// ReplaceImages replaces image names in a list of manifests.
func (l *ManifestList) ReplaceImages(builds []build.Artifact) (ManifestList, error) {
	source := newImageNameSource(builds)
	replacer := &multiReplacer{
		any: []Replacer{
			&imageFieldReplacer{
				source: source,
			},
			&imageEnvReplacer{
				source: source,
			},
		},
	}

	updated, err := l.Visit(replacer)
	if err != nil {
		return nil, errors.Wrap(err, "replacing images")
	}

	source.Check()
	logrus.Debugln("manifests with tagged images", updated.String())

	return updated, nil
}

type ReplacerSource interface {
	NewValue(image string) (bool, string)
}

type imageNameSource struct {
	tagsByImageName map[string]string
	found           map[string]bool
}

func newImageNameSource(builds []build.Artifact) *imageNameSource {
	tagsByImageName := make(map[string]string)
	for _, build := range builds {
		tagsByImageName[build.ImageName] = build.Tag
	}

	return &imageNameSource{
		tagsByImageName: tagsByImageName,
		found:           make(map[string]bool),
	}
}

func (r *imageNameSource) Check() {
	for imageName := range r.tagsByImageName {
		if !r.found[imageName] {
			warner.Warnf("image [%s] is not used by the deployment", imageName)
		}
	}
}

func (r *imageNameSource) NewValue(image string) (bool, string) {
	parsed, err := docker.ParseReference(image)
	if err != nil {
		warner.Warnf("Couldn't parse image: %s", image)
		return false, image
	}

	if tag, present := r.tagsByImageName[parsed.BaseName]; present {
		if parsed.FullyQualified {
			if tag == image {
				r.found[parsed.BaseName] = true
			}
		} else {
			r.found[parsed.BaseName] = true
			return true, tag
		}
	}

	return false, image
}

type imageFieldReplacer struct {
	source ReplacerSource
}

func (r *imageFieldReplacer) Matches(key string) bool {
	return key == "image"
}

func (r *imageFieldReplacer) NewValue(key string, old interface{}) (bool, interface{}) {
	return r.source.NewValue(old.(string))
}

type imageEnvReplacer struct {
	source ReplacerSource
}

func (r *imageEnvReplacer) Matches(key string) bool {
	return key == "env"
}

func (r *imageEnvReplacer) NewValue(key string, old interface{}) (bool, interface{}) {
	if old.([]interface{}) == nil {
		return false, old
	}

	replaced := false
	for _, elt := range old.([]interface{}) {
		if elt.(map[interface{}]interface{}) == nil {
			break
		}

		envVarEntry := elt.(map[interface{}]interface{})
		envVarName := envVarEntry["name"].(string)
		if envVarName == "" {
			break
		}

		if strings.HasSuffix(envVarName, "_IMAGE") {
			image := envVarEntry["value"].(string)
			if present, new := r.source.NewValue(image); present {
				envVarEntry["value"] = new
				replaced = true
			}
		}
	}

	return replaced, old
}

type multiReplacer struct {
	any []Replacer
}

func (r *multiReplacer) matchingReplacer(key string) Replacer {
	for _, replacer := range r.any {
		if replacer.Matches(key) {
			return replacer
		}
	}
	return nil
}

func (r *multiReplacer) Matches(key string) bool {
	return r.matchingReplacer(key) != nil
}

func (r *multiReplacer) NewValue(key string, old interface{}) (bool, interface{}) {
	replacer := r.matchingReplacer(key)
	if replacer == nil {
		return false, old
	}

	return replacer.NewValue(key, old)
}
