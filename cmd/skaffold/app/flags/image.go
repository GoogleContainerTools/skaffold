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

package flags

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// Images describes a flag which contains a list of image names
type Images struct {
	images []image
	usage  string
}

type image struct {
	name     string
	artifact *build.Artifact
}

// GetSlice Implements GetSlice() method for pflag SliceValue interface and
// returns a slice image names.
func (i *Images) GetSlice() []string {
	names := make([]string, len(i.images))
	for i, image := range i.images {
		names[i] = image.name
	}
	return names
}

// Append Implements Append() method for pflag SliceValue interface
func (i *Images) Append(value string) error {
	a, err := convertImageToArtifact(value)
	if err != nil {
		return err
	}
	i.images = append(i.images, image{name: value, artifact: a})
	return nil
}

// Replace Implements Replace() method for pflag SliceValue interface
func (i *Images) Replace(images []string) error {
	newImages := make([]image, len(images))
	for _, value := range images {
		a, err := convertImageToArtifact(value)
		if err != nil {
			return err
		}
		newImages = append(newImages, image{name:value, artifact:a})
		i.images = newImages
	}
	return nil
}

// Artifacts returns an artifact representation for the corresponding image
func (i *Images) Artifacts() []build.Artifact {
	var artifacts []build.Artifact

	for _, image := range i.images {
		artifacts = append(artifacts, *image.artifact)
	}

	return artifacts
}

// NewEmptyImages returns a new nil Images list.
func NewEmptyImages(usage string) *Images {
	return &Images{
		images: []image{},
		usage:  usage,
	}
}

func convertImageToArtifact(value string) (*build.Artifact, error) {
	if value == "" {
		return nil, errors.New("cannot add an empty image value")
	}
	parsed, err := docker.ParseReference(value)
	if err != nil {
		return nil, err
	}
	return &build.Artifact{
		ImageName: parsed.BaseName,
		Tag:       value,
	}, nil
}
