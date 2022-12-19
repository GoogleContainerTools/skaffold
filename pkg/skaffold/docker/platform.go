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
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	spec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// GetPlatforms returns the platforms of the provided image.
func GetPlatforms(image string) ([]spec.Platform, error) {
	i, err := getImage(image)
	if err != nil {
		return nil, err
	}

	mt, err := i.MediaType()
	if err != nil {
		return nil, err
	}
	var p []spec.Platform
	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		ix, ok := i.(v1.ImageIndex)
		if !ok {
			return nil, fmt.Errorf("failed to interpret %q as index: %v", image, i)
		}
		manifests, err := ix.IndexManifest()
		if err != nil {
			return nil, err
		}
		for _, m := range manifests.Manifests {
			if m.Platform == nil {
				continue
			}
			p = append(p, util.ConvertFromV1Platform(*m.Platform))
		}
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		im, ok := i.(v1.Image)
		if !ok {
			return nil, fmt.Errorf("failed to interpret %q as image: %v", image, im)
		}
		cf, err := im.ConfigFile()
		if err != nil {
			return nil, err
		}
		p = append(p, spec.Platform{OS: cf.OS, Architecture: cf.Architecture})
	default:
		return nil, fmt.Errorf("image media type: %s", mt)
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func getImage(image string) (ImageRef, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(primaryKeychain))
	if err != nil {
		return nil, err
	}
	if desc.MediaType.IsIndex() {
		return desc.ImageIndex()
	}
	return desc.Image()
}

// ImageRef represents a v1.Image or v1.ImageIndex.
type ImageRef interface {
	MediaType() (types.MediaType, error)
}
