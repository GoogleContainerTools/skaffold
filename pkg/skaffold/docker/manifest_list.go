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
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// for testing
var (
	mutateAppendManifest = mutate.AppendManifests
	remoteWriteIndex     = remote.WriteIndex
)

// Describes the result of an image build.
type SinglePlatformImage struct {
	// Platform (OS + architecture) associated with the image built.
	Platform *v1.Platform

	// Name of the image built.
	Image string
}

// CreateManifestList returns a manifest list that contains the given images.
func CreateManifestList(ctx context.Context, images []SinglePlatformImage, targetTag string) (string, error) {
	adds := make([]mutate.IndexAddendum, len(images))

	for i, image := range images {
		ref, err := name.ParseReference(image.Image, name.WeakValidation)
		if err != nil {
			return "", err
		}

		img, err := remoteImage(ref, remote.WithAuthFromKeychain(primaryKeychain))
		if err != nil {
			return "", err
		}

		adds[i] = mutate.IndexAddendum{
			Add: img,
			Descriptor: v1.Descriptor{
				Platform: image.Platform,
			},
		}
	}
	idx := mutateAppendManifest(mutate.IndexMediaType(empty.Index, types.DockerManifestList), adds...)
	targetRef, err := name.ParseReference(targetTag, name.WeakValidation)
	if err != nil {
		return "", err
	}

	err = remoteWriteIndex(targetRef, idx, remote.WithAuthFromKeychain(primaryKeychain))
	if err != nil {
		return "", err
	}

	h, err := idx.Digest()
	if err != nil {
		return "", err
	}

	dig := h.String()
	log.Entry(ctx).Printf("Created ManifestList for image %s. Digest: %s\n", targetRef, dig)
	parsed, err := ParseReference(targetTag)
	if err != nil {
		return "", err
	}

	// TODO: manifestlists are only supported in a container registry so we
	// return the fully qualified image name with tag and digest. When the local
	// docker daemon can support multi-platform images, we'll have to return the
	// image with imageID.
	return fmt.Sprintf("%s:%s@%s", parsed.BaseName, parsed.Tag, dig), nil
}
