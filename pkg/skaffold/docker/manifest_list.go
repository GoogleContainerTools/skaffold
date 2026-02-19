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

		img, err := fetchSinglePlatformImage(ref, image.Platform)
		if err != nil {
			return "", fmt.Errorf("fetching image %s for platform %v: %w", image.Image, image.Platform, err)
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

// fetchSinglePlatformImage retrieves a single-platform image from the registry.
// When docker build --push runs with BuildKit enabled for cross-platform builds,
// BuildKit may wrap the pushed image in an OCI Index that also contains a provenance
// attestation manifest. In that case remote.Image() fails with
// "no child with platform <host> in index" because remote.Image uses the current
// host platform as the selector, which may not match the intended build platform.
//
// Strategy: try remoteImage first (the common, fast path). If it fails (the reference
// points to an OCI Index), fall back to remoteIndex and extract the platform-specific
// child image from the index.
func fetchSinglePlatformImage(ref name.Reference, platform *v1.Platform) (v1.Image, error) {
	img, err := remoteImage(ref, remote.WithAuthFromKeychain(primaryKeychain))
	if err == nil {
		return img, nil
	}

	// remoteImage failed — the reference may be an OCI Index wrapping the real image
	// plus a BuildKit attestation manifest. Try fetching as an index and extracting
	// the intended platform image.
	idx, idxErr := remoteIndex(ref, remote.WithAuthFromKeychain(primaryKeychain))
	if idxErr != nil {
		// Neither worked — return the original error.
		return nil, err
	}

	return extractPlatformImageFromIndex(idx, platform)
}

// extractPlatformImageFromIndex finds and returns the image for the given platform
// within a manifest index, skipping BuildKit attestation manifests.
func extractPlatformImageFromIndex(idx v1.ImageIndex, platform *v1.Platform) (v1.Image, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("getting index manifest: %w", err)
	}

	for _, m := range manifest.Manifests {
		// Skip attestation manifests (BuildKit marks them with os=unknown or via annotation).
		if m.Platform != nil && m.Platform.OS == "unknown" {
			continue
		}
		if m.Annotations["vnd.docker.reference.type"] == "attestation-manifest" {
			continue
		}
		if platform != nil && m.Platform != nil {
			if m.Platform.OS == platform.OS && m.Platform.Architecture == platform.Architecture {
				return idx.Image(m.Digest)
			}
			continue
		}
		// No platform filter — return the first non-attestation image.
		if m.MediaType == types.OCIManifestSchema1 || m.MediaType == types.DockerManifestSchema2 || m.MediaType == "" {
			return idx.Image(m.Digest)
		}
	}

	return nil, fmt.Errorf("no suitable image found in index for platform %v", platform)
}
