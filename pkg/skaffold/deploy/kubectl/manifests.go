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
	"bytes"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// for testing
var warner Warner = &logrusWarner{}

// ManifestList is a list of yaml manifests.
type ManifestList [][]byte

func (l *ManifestList) String() string {
	var str string
	for i, manifest := range *l {
		if i != 0 {
			str += "\n---\n"
		}
		str += string(bytes.TrimSpace(manifest))
	}
	return str
}

// Append appends the yaml manifests defined in the given buffer.
func (l *ManifestList) Append(buf []byte) {
	parts := bytes.Split(buf, []byte("\n---"))
	for _, part := range parts {
		*l = append(*l, part)
	}
}

// Diff computes the list of manifests that have changed.
func (l *ManifestList) Diff(latest ManifestList) ManifestList {
	if l == nil {
		return latest
	}

	oldManifests := map[string]bool{}
	for _, oldManifest := range *l {
		oldManifests[string(oldManifest)] = true
	}

	var updated ManifestList

	for _, manifest := range latest {
		if !oldManifests[string(manifest)] {
			updated = append(updated, manifest)
		}
	}

	return updated
}

// Reader returns a reader on the raw yaml descriptors.
func (l *ManifestList) Reader() io.Reader {
	return strings.NewReader(l.String())
}

type replacement struct {
	tag   string
	found bool
}

// ReplaceImages replaces the image names in the manifests.
func (l *ManifestList) ReplaceImages(builds []build.Artifact) (ManifestList, error) {
	replacements := map[string]*replacement{}
	for _, build := range builds {
		replacements[build.ImageName] = &replacement{
			tag: build.Tag,
		}
	}

	var updatedManifests ManifestList

	for _, manifest := range *l {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		recursiveReplaceImage(m, replacements)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}

		updatedManifests = append(updatedManifests, updatedManifest)
	}

	for name, replacement := range replacements {
		if !replacement.found {
			warner.Warnf("image [%s] is not used by the deployment", name)
		}
	}

	logrus.Debugln("manifests with tagged images", updatedManifests.String())

	return updatedManifests, nil
}

func recursiveReplaceImage(i interface{}, replacements map[string]*replacement) {
	switch t := i.(type) {
	case []interface{}:
		for _, v := range t {
			recursiveReplaceImage(v, replacements)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			if k.(string) != "image" {
				recursiveReplaceImage(v, replacements)
				continue
			}

			image := v.(string)
			parsed, err := docker.ParseReference(image)
			if err != nil {
				warner.Warnf("Couldn't parse image: %s", v)
				continue
			}

			if img, present := replacements[parsed.BaseName]; present {
				if parsed.FullyQualified {
					if img.tag == image {
						img.found = true
					}
				} else {
					t[k] = img.tag
					img.found = true
				}
			}
		}
	}
}
