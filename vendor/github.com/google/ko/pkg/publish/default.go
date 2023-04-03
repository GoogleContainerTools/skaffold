// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package publish

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/pkg/oci"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/walk"

	"github.com/google/ko/pkg/build"
)

// defalt is intentionally misspelled to avoid keyword collision (and drive Jon nuts).
type defalt struct {
	base      string
	t         http.RoundTripper
	userAgent string
	auth      authn.Authenticator
	namer     Namer
	tags      []string
	tagOnly   bool
	insecure  bool
}

// Option is a functional option for NewDefault.
type Option func(*defaultOpener) error

type defaultOpener struct {
	base      string
	t         http.RoundTripper
	userAgent string
	auth      authn.Authenticator
	namer     Namer
	tags      []string
	tagOnly   bool
	insecure  bool
}

// Namer is a function from a supported import path to the portion of the resulting
// image name that follows the "base" repository name.
type Namer func(string, string) string

// identity is the default namer, so import paths are affixed as-is under the repository
// name for maximum clarity, e.g.
//   gcr.io/foo/github.com/bar/baz/cmd/blah
//   ^--base--^ ^-------import path-------^
func identity(base, in string) string { return path.Join(base, in) }

// As some registries do not support pushing an image by digest, the default tag for pushing
// is the 'latest' tag.
var defaultTags = []string{"latest"}

func (do *defaultOpener) Open() (Interface, error) {
	if do.tagOnly {
		if len(do.tags) != 1 {
			return nil, errors.New("must specify exactly one tag to resolve images into tag-only references")
		}
		if do.tags[0] == defaultTags[0] {
			return nil, errors.New("latest tag cannot be used in tag-only references")
		}
	}

	return &defalt{
		base:      do.base,
		t:         do.t,
		userAgent: do.userAgent,
		auth:      do.auth,
		namer:     do.namer,
		tags:      do.tags,
		tagOnly:   do.tagOnly,
		insecure:  do.insecure,
	}, nil
}

// NewDefault returns a new publish.Interface that publishes references under the provided base
// repository using the default keychain to authenticate and the default naming scheme.
func NewDefault(base string, options ...Option) (Interface, error) {
	do := &defaultOpener{
		base:      base,
		t:         http.DefaultTransport,
		userAgent: "ko",
		auth:      authn.Anonymous,
		namer:     identity,
		tags:      defaultTags,
	}

	for _, option := range options {
		if err := option(do); err != nil {
			return nil, err
		}
	}

	return do.Open()
}

func pushResult(ctx context.Context, tag name.Tag, br build.Result, opt []remote.Option) error {
	mt, err := br.MediaType()
	if err != nil {
		return err
	}

	// writePeripherals implements walk.Fn
	writePeripherals := func(ctx context.Context, se oci.SignedEntity) error {
		ociOpts := []ociremote.Option{ociremote.WithRemoteOptions(opt...)}

		// Respect COSIGN_REPOSITORY
		targetRepoOverride, err := ociremote.GetEnvTargetRepository()
		if err != nil {
			return err
		}
		if (targetRepoOverride != name.Repository{}) {
			ociOpts = append(ociOpts, ociremote.WithTargetRepository(targetRepoOverride))
		}
		h, err := se.(interface{ Digest() (v1.Hash, error) }).Digest()
		if err != nil {
			return err
		}

		// TODO(mattmoor): We should have a WriteSBOM helper upstream.
		digest := tag.Context().Digest(h.String()) // Don't *get* the tag, we know the digest
		ref, err := ociremote.SBOMTag(digest, ociOpts...)
		if err != nil {
			return err
		}
		if f, err := se.Attachment("sbom"); err != nil {
			// Some levels (e.g. the index) may not have an SBOM,
			// just like some levels may not have signatures/attestations.
		} else if err := remote.Write(ref, f, opt...); err != nil {
			return fmt.Errorf("writing sbom: %w", err)
		} else {
			log.Printf("Published SBOM %v", ref)
		}

		// TODO(mattmoor): Don't enable this until we start signing or it
		// will publish empty signatures!
		// if err := ociremote.WriteSignatures(tag.Context(), se, ociOpts...); err != nil {
		// 	return err
		// }

		// TODO(mattmoor): Are there any attestations we want to write?
		// if err := ociremote.WriteAttestations(tag.Context(), se, ociOpts...); err != nil {
		// 	return err
		// }
		return nil
	}

	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		idx, ok := br.(v1.ImageIndex)
		if !ok {
			return fmt.Errorf("failed to interpret result as index: %v", br)
		}
		if sii, ok := idx.(oci.SignedImageIndex); ok {
			if err := walk.SignedEntity(ctx, sii, writePeripherals); err != nil {
				return err
			}
		}
		return remote.WriteIndex(tag, idx, opt...)
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		img, ok := br.(v1.Image)
		if !ok {
			return fmt.Errorf("failed to interpret result as image: %v", br)
		}
		if si, ok := img.(oci.SignedImage); ok {
			if err := writePeripherals(ctx, si); err != nil {
				return err
			}
		}
		return remote.Write(tag, img, opt...)
	default:
		return fmt.Errorf("result image media type: %s", mt)
	}
}

// Publish implements publish.Interface
func (d *defalt) Publish(ctx context.Context, br build.Result, s string) (name.Reference, error) {
	s = strings.TrimPrefix(s, build.StrictScheme)
	// https://github.com/google/go-containerregistry/issues/212
	s = strings.ToLower(s)

	ro := []remote.Option{remote.WithAuth(d.auth), remote.WithTransport(d.t), remote.WithContext(ctx), remote.WithUserAgent(d.userAgent)}
	no := []name.Option{}
	if d.insecure {
		no = append(no, name.Insecure)
	}

	for i, tagName := range d.tags {
		tag, err := name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), tagName), no...)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			log.Printf("Publishing %v", tag)
			if err := pushResult(ctx, tag, br, ro); err != nil {
				return nil, err
			}
		} else {
			log.Printf("Tagging %v", tag)
			if err := remote.Tag(tag, br, ro...); err != nil {
				return nil, err
			}
		}
	}

	if d.tagOnly {
		// We have already validated that there is a single tag (not latest).
		tag, err := name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), d.tags[0]))
		if err != nil {
			return nil, err
		}
		return &tag, nil
	}

	h, err := br.Digest()
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("%s@%s", d.namer(d.base, s), h)
	if len(d.tags) == 1 && d.tags[0] != defaultTags[0] {
		// If a single tag is explicitly set (not latest), then this
		// is probably a release, so include the tag in the reference.
		ref = fmt.Sprintf("%s:%s@%s", d.namer(d.base, s), d.tags[0], h)
	}
	dig, err := name.NewDigest(ref)
	if err != nil {
		return nil, err
	}
	log.Printf("Published %v", dig)
	return &dig, nil
}

func (d *defalt) Close() error {
	return nil
}
