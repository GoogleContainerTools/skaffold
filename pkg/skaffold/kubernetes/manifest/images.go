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
	"strings"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

const imageField = "image"

type ResourceSelectorImages struct {
	allowlist map[apimachinery.GroupKind]latest.ResourceFilter
	denylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func NewResourceSelectorImages(allowlist map[apimachinery.GroupKind]latest.ResourceFilter, denylist map[apimachinery.GroupKind]latest.ResourceFilter) *ResourceSelectorImages {
	return &ResourceSelectorImages{
		allowlist: allowlist,
		denylist:  denylist,
	}
}

func (rsi *ResourceSelectorImages) allowByGroupKind(gk apimachinery.GroupKind) bool {
	if _, allowed := rsi.allowlist[gk]; allowed {
		// TODO(aaron-prindle) see if it makes sense to make this only use the allowlist...
		if rf, disallowed := rsi.denylist[gk]; disallowed {
			for _, s := range rf.Labels {
				if s == ".*" {
					return false
				}
			}
			for _, s := range rf.Image {
				if s == ".*" {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (rsi *ResourceSelectorImages) allowByNavpath(gk apimachinery.GroupKind, navpath string, k string) (string, bool) {
	matchedConfigConnectorImage := false

	for _, w := range ConfigConnectorResourceSelector {
		if k == imageField && w.Matches(gk.Group, gk.Kind) {
			matchedConfigConnectorImage = true
			break
		}
	}

	if rf, ok := rsi.denylist[gk]; ok {
		for _, denypath := range rf.Image {
			if denypath == ".*" {
				return "", false
			}
			if navpath == denypath {
				return "", false
			}
		}
	}

	if rf, ok := rsi.allowlist[gk]; ok {
		matchedConfigConnectorImage = false

		for _, allowpath := range rf.Image {
			if allowpath == ".*" && k == imageField {
				return "", true
			}
			if navpath == allowpath {
				return "", true
			}
		}
	}
	return "", matchedConfigConnectorImage
}

// GetImages gathers a map of base image names to the image with its tag
func (l *ManifestList) GetImages(rs ResourceSelector) ([]graph.Artifact, error) {
	s := &imageSaver{}
	_, err := l.Visit(s, rs)
	return s.Images, parseImagesInManifestErr(err)
}

type imageSaver struct {
	Images []graph.Artifact
}

func (is *imageSaver) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if k != imageField {
		return true
	}

	image, ok := v.(string)
	if !ok {
		return true
	}
	parsed, err := docker.ParseReference(image)
	if err != nil {
		log.Entry(context.TODO()).Debugf("Couldn't parse image [%s]: %s", image, err.Error())
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
func (l *ManifestList) ReplaceImages(ctx context.Context, builds []graph.Artifact, rs ResourceSelector) (ManifestList, error) {
	return l.replaceImages(ctx, builds, selectLocalManifestImages, rs)
}

// ReplaceRemoteManifestImages replaces all image names in a list containing remote manifests.
// This will even override images referenced by digest or with a different repository
func (l *ManifestList) ReplaceRemoteManifestImages(ctx context.Context, builds []graph.Artifact, rs ResourceSelector) (ManifestList, error) {
	return l.replaceImages(ctx, builds, selectRemoteManifestImages, rs)
}

func (l *ManifestList) replaceImages(ctx context.Context, builds []graph.Artifact, selector imageSelector, rs ResourceSelector) (ManifestList, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "ReplaceImages", map[string]string{
		"manifestEntries":   strconv.Itoa(len(*l)),
		"numImagesReplaced": strconv.Itoa(len(builds)),
	})
	defer endTrace()

	replacer := newImageReplacer(builds, selector)

	updated, err := l.Visit(replacer, rs)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, replaceImageErr(err)
	}

	replacer.Check()
	log.Entry(ctx).Debug("manifests with tagged images:", updated.String())

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

func (r *imageReplacer) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if _, ok := rs.allowByNavpath(gk, navpath, k); !ok {
		return true
	}

	image, ok := v.(string)
	if !ok {
		return true
	}

	parsed, err := docker.ParseReference(image)
	if err != nil {
		// this is a hack to properly support `imageStrategy=helm` & `imageStrategy=helm+explicitRegistry` from Skaffold v1.X.X
		// which have the form:
		// helm - image: "{{.Values.image.repository}}:{{.Values.image.tag}}"
		// helm+explicitRegistry - image: "{{.Values.image.registry}}/{{.Values.image.repository}}:{{.Values.image.tag}}"
		// when the artifact name has a fully qualified path - gcr.io/example-repo/skaffold-helm-image
		// works by looking for intermediate helm replacement of the form image:
		// helm - image: <artifactName>/<artifactName>:<artifactName>
		// helm+explicitRegistry - image: <artifactName>:<artifactName>
		// and treating that as just <artifact> by modifying the parsed representation.
		tagSplit := strings.Split(image, ":")
		if len(tagSplit) == 2 && strings.HasPrefix(image, tagSplit[1]) {
			if _, present := r.tagsByImageName[tagSplit[1]]; present {
				parsed = &docker.ImageReference{
					BaseName: tagSplit[1],
				}
			}
		} else {
			log.Entry(context.TODO()).Debugf("Couldn't parse image [%s]: %s", image, err.Error())
			return false
		}
	}
	// this is a hack to properly support `imageStrategy=helm+explicitRegistry` from Skaffold v1.X.X which has the form:
	// image: "{{.Values.image.registry}}/{{.Values.image.repository}}:{{.Values.image.tag}}"
	// when the artifact name is not fully qualified - skaffold-helm-image
	// works by looking for intermediate helm replacement of the form image:
	// image: <artifactName>/<artifactName>:<artifactName>
	// and treating that as just <artifact> by modifying the parsed representation.
	if parsed != nil && parsed.Domain != "" && parsed.Domain == parsed.Repo {
		if _, present := r.tagsByImageName[parsed.Repo]; present {
			parsed.BaseName = parsed.Repo
		}
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
			log.Entry(context.TODO()).Debugf("image [%s] is not used by the current deployment", imageName)
		}
	}
}

// imageSelector represents a strategy for matching the container `image` defined in a kubernetes manifest with the correct skaffold artifact.
type imageSelector func(tagsByImageName map[string]string, image *docker.ImageReference) (imageName, tag string, valid bool)

func selectLocalManifestImages(tagsByImageName map[string]string, image *docker.ImageReference) (string, string, bool) {
	if image == nil {
		return "", "", false
	}

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
