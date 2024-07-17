// Copyright 2018 ko Build Authors All Rights Reserved.
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
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/walk"
	"golang.org/x/sync/errgroup"

	"github.com/google/ko/pkg/build"
)

// defalt is intentionally misspelled to avoid keyword collision (and drive Jon nuts).
type defalt struct {
	base     string
	namer    Namer
	tags     []string
	tagOnly  bool
	insecure bool
	jobs     int

	pusher *remote.Pusher
	oopt   []ociremote.Option
}

// Option is a functional option for NewDefault.
type Option func(*defaultOpener) error

type defaultOpener struct {
	base      string
	t         http.RoundTripper
	userAgent string
	keychain  authn.Keychain
	namer     Namer
	tags      []string
	tagOnly   bool
	insecure  bool
	ropt      []remote.Option
	jobs      int
}

// Namer is a function from a supported import path to the portion of the resulting
// image name that follows the "base" repository name.
type Namer func(string, string) string

// identity is the default namer, so import paths are affixed as-is under the repository
// name for maximum clarity, e.g.
//
//	gcr.io/foo/github.com/bar/baz/cmd/blah
//	^--base--^ ^-------import path-------^
func identity(base, in string) string { return path.Join(base, in) }

// As some registries do not support pushing an image by digest, the default tag for pushing
// is the 'latest' tag.
const latestTag = "latest"

func (do *defaultOpener) Open() (Interface, error) {
	if do.tagOnly {
		if len(do.tags) != 1 {
			return nil, errors.New("must specify exactly one tag to resolve images into tag-only references")
		}
		if do.tags[0] == latestTag {
			return nil, errors.New("latest tag cannot be used in tag-only references")
		}
	}

	pusher, err := remote.NewPusher(do.ropt...)
	if err != nil {
		return nil, err
	}

	oopt := []ociremote.Option{ociremote.WithRemoteOptions(do.ropt...)}

	// Respect COSIGN_REPOSITORY
	targetRepoOverride, err := ociremote.GetEnvTargetRepository()
	if err != nil {
		return nil, err
	}
	if (targetRepoOverride != name.Repository{}) {
		oopt = append(oopt, ociremote.WithTargetRepository(targetRepoOverride))
	}

	return &defalt{
		base:     do.base,
		namer:    do.namer,
		tags:     do.tags,
		tagOnly:  do.tagOnly,
		insecure: do.insecure,
		jobs:     do.jobs,
		pusher:   pusher,
		oopt:     oopt,
	}, nil
}

// NewDefault returns a new publish.Interface that publishes references under the provided base
// repository using the default keychain to authenticate and the default naming scheme.
func NewDefault(base string, options ...Option) (Interface, error) {
	do := &defaultOpener{
		base:      base,
		t:         remote.DefaultTransport,
		userAgent: "ko",
		keychain:  authn.DefaultKeychain,
		namer:     identity,
		tags:      []string{latestTag},
	}

	for _, option := range options {
		if err := option(do); err != nil {
			return nil, err
		}
	}

	do.ropt = []remote.Option{remote.WithAuthFromKeychain(do.keychain), remote.WithTransport(do.t), remote.WithUserAgent(do.userAgent)}
	if do.jobs == 0 {
		do.jobs = runtime.GOMAXPROCS(0)
		do.ropt = append(do.ropt, remote.WithJobs(do.jobs))
	}

	return do.Open()
}

func (d *defalt) pushResult(ctx context.Context, tag name.Tag, br build.Result) error {
	mt, err := br.MediaType()
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(d.jobs)

	g.Go(func() error {
		return d.pusher.Push(ctx, tag, br)
	})

	// writePeripherals implements walk.Fn
	writePeripherals := func(ctx context.Context, se oci.SignedEntity) error {
		h, err := se.(interface{ Digest() (v1.Hash, error) }).Digest()
		if err != nil {
			return err
		}

		// TODO(mattmoor): We should have a WriteSBOM helper upstream.
		digest := tag.Context().Digest(h.String()) // Don't *get* the tag, we know the digest
		ref, err := ociremote.SBOMTag(digest, d.oopt...)
		if err != nil {
			return err
		}
		f, err := se.Attachment("sbom")
		if err != nil {
			// Some levels (e.g. the index) may not have an SBOM,
			// just like some levels may not have signatures/attestations.
			return nil
		}

		g.Go(func() error {
			if err := d.pusher.Push(ctx, ref, f); err != nil {
				return fmt.Errorf("writing sbom: %w", err)
			}

			log.Printf("Published SBOM %v", ref)
			return nil
		})

		// TODO(mattmoor): Don't enable this until we start signing or it
		// will publish empty signatures!
		// if err := ociremote.WriteSignatures(tag.Context(), se, oopt...); err != nil {
		// 	return err
		// }

		// TODO(mattmoor): Are there any attestations we want to write?
		// if err := ociremote.WriteAttestations(tag.Context(), se, oopt...); err != nil {
		// 	return err
		// }

		return nil
	}

	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		if sii, ok := br.(oci.SignedImageIndex); ok {
			if err := walk.SignedEntity(ctx, sii, writePeripherals); err != nil {
				return err
			}
		}
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		if si, ok := br.(oci.SignedImage); ok {
			if err := writePeripherals(ctx, si); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("result image media type: %s", mt)
	}

	return g.Wait()
}

// Publish implements publish.Interface
func (d *defalt) Publish(ctx context.Context, br build.Result, s string) (name.Reference, error) {
	s = strings.TrimPrefix(s, build.StrictScheme)
	// https://github.com/google/go-containerregistry/issues/212
	s = strings.ToLower(s)

	no := []name.Option{}
	if d.insecure {
		no = append(no, name.Insecure)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(d.jobs)
	for i, tagName := range d.tags {
		tag, err := name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), tagName), no...)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			log.Printf("Publishing %v", tag)
			g.Go(func() error {
				return d.pushResult(ctx, tag, br)
			})
		} else {
			g.Go(func() error {
				log.Printf("Tagging %v", tag)
				return d.pusher.Push(ctx, tag, br)
			})
		}
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	if d.tagOnly {
		// We have already validated that there is a single tag (not latest).
		return name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), d.tags[0]))
	}

	h, err := br.Digest()
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("%s@%s", d.namer(d.base, s), h)
	if len(d.tags) == 1 && d.tags[0] != latestTag {
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
