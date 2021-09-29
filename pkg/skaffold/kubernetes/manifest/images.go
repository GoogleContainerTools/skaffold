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
		logrus.Debugf("Couldn't parse image [%s]: %s", image, err.Error())
		return false
	}

	is.Images = append(is.Images, graph.Artifact{
		Tag:       image,
		ImageName: parsed.BaseName,
	})
	return false
}

// ReplaceImages replaces image names in a list of manifests.
// It doesn't replace images that are referenced by digest.
func (l *ManifestList) ReplaceImages(ctx context.Context, builds []graph.Artifact) (ManifestList, error) {
	return l.replaceImages(ctx, builds, selectLocalManifestImages)
}

// ReplaceRemoteManifestImages replaces all image names in a list containing remote manifests.
// This will even override images referenced by digest or with a different repository
func (l *ManifestList) ReplaceRemoteManifestImages(ctx context.Context, builds []graph.Artifact) (ManifestList, error) {
	return l.replaceImages(ctx, builds, selectRemoteManifestImages)
}

func (l *ManifestList) replaceImages(ctx context.Context, builds []graph.Artifact, selector imageSelector) (ManifestList, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "ReplaceImages", map[string]string{
		"manifestEntries":   strconv.Itoa(len(*l)),
		"numImagesReplaced": strconv.Itoa(len(builds)),
	})
	defer endTrace()

	replacer := newImageReplacer(builds, selector)

	updated, err := l.Visit(replacer)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, replaceImageErr(err)
	}

	replacer.Check()
	logrus.Debugln("manifests with tagged images:", updated.String())

	return updated, nil
}

type imageReplacer struct {
	tagsByImageName map[string]string
	found           map[string]bool
	selector        imageSelector
}

func newImageReplacer(builds []graph.Artifact, selector imageSelector) *imageReplacer {
	tagsByImageName := make(map[string]string)
	for _, build := range builds {
		imageName := docker.SanitizeImageName(build.ImageName)
		tagsByImageName[imageName] = build.Tag
	}

	return &imageReplacer{
		tagsByImageName: tagsByImageName,
		found:           make(map[string]bool),
		selector:        selector,
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
		logrus.Debugf("Couldn't parse image [%s]: %s", image, err.Error())
		return false
	}
	if imageName, tag, selected := r.selector(r.tagsByImageName, parsed); selected {
		r.found[imageName] = true
		o[k] = tag
	}
	return false
}

func (r *imageReplacer) Check() {
	for imageName := range r.tagsByImageName {
		if !r.found[imageName] {
			logrus.Debugf("image [%s] is not used by the current deployment", imageName)
		}
	}
}

// imageSelector represents a strategy for matching the container `image` defined in a kubernetes manifest with the correct skaffold artifact.
type imageSelector func(tagsByImageName map[string]string, image *docker.ImageReference) (imageName, tag string, valid bool)

func selectLocalManifestImages(tagsByImageName map[string]string, image *docker.ImageReference) (string, string, bool) {
	// Leave images referenced by digest as they are
	if image.Digest != "" {
		return "", "", false
	}
	// local manifest mentions artifact `imageName` directly, so `imageName` is parsed into `image.BaseName`
	tag, present := tagsByImageName[image.BaseName]
	return image.BaseName, tag, present
}

func selectRemoteManifestImages(tagsByImageName map[string]string, image *docker.ImageReference) (string, string, bool) {
	// if manifest mentions `imageName` directly then `imageName` is parsed into `image.BaseName`
	if tag, present := tagsByImageName[image.BaseName]; present {
		return image.BaseName, tag, present
	}
	// if manifest mentions image with repository then `imageName` is parsed into `image.Name`
	tag, present := tagsByImageName[image.Name]
	return image.Name, tag, present
}
