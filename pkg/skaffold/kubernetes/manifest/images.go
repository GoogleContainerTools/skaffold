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

package manifest

import (
	"context"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

// GetImages gathers a map of base image names to the image with its tag
func (l *ManifestList) GetImages() ([]graph.Artifact, error) {
	s := &imageSaver{}
	_, err := l.Visit(s)
	return s.Images, parseImagesInManifestErr(err)
}

type imageSaver struct {
	Images []graph.Artifact
}

func (is *imageSaver) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "image" {
		return true
	}

	image, ok := v.(string)
	if !ok {
		return true
	}
	parsed, err := docker.ParseReference(image)
	if err != nil {
		warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
		return false
	}

	is.Images = append(is.Images, graph.Artifact{
		Tag:       image,
		ImageName: parsed.BaseName,
	})
	return false
}

// ReplaceImages replaces image names in a list of manifests.
func (l *ManifestList) ReplaceImages(ctx context.Context, log *logrus.Logger, builds []graph.Artifact) (ManifestList, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "ReplaceImages", map[string]string{
		"manifestEntries":   strconv.Itoa(len(*l)),
		"numImagesReplaced": strconv.Itoa(len(builds)),
	})
	defer endTrace()

	replacer := newImageReplacer(builds)

	updated, err := l.Visit(replacer)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, replaceImageErr(err)
	}

	replacer.Check(log)
	log.Debugln("manifests with tagged images:", updated.String())

	return updated, nil
}

type imageReplacer struct {
	tagsByImageName map[string]string
	found           map[string]bool
}

func newImageReplacer(builds []graph.Artifact) *imageReplacer {
	tagsByImageName := make(map[string]string)
	for _, build := range builds {
		imageName := docker.SanitizeImageName(build.ImageName)
		tagsByImageName[imageName] = build.Tag
	}

	return &imageReplacer{
		tagsByImageName: tagsByImageName,
		found:           make(map[string]bool),
	}
}

func (r *imageReplacer) Visit(o map[string]interface{}, k string, v interface{}) bool {
	if k != "image" {
		return true
	}

	image, ok := v.(string)
	if !ok {
		return true
	}
	parsed, err := docker.ParseReference(image)
	if err != nil {
		warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
		return false
	}
	// Leave images referenced by digest as they are
	if parsed.Digest != "" {
		return false
	}
	if tag, present := r.tagsByImageName[parsed.BaseName]; present {
		// Apply new image tag
		r.found[parsed.BaseName] = true
		o[k] = tag
	}
	return false
}

func (r *imageReplacer) Check(log *logrus.Logger) {
	for imageName := range r.tagsByImageName {
		if !r.found[imageName] {
			log.Debugf("image [%s] is not used by the current deployment", imageName)
		}
	}
}
