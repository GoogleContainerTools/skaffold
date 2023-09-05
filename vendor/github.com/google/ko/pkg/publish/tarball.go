// Copyright 2020 ko Build Authors All Rights Reserved.
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
	"fmt"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/ko/pkg/build"
)

type tar struct {
	file  string
	base  string
	namer Namer
	tags  []string
	refs  map[name.Reference]v1.Image
}

// NewTarball returns a new publish.Interface that saves images to a tarball.
func NewTarball(file, base string, namer Namer, tags []string) Interface {
	return &tar{
		file:  file,
		base:  base,
		namer: namer,
		tags:  tags,
		refs:  make(map[name.Reference]v1.Image),
	}
}

// Publish implements publish.Interface.
func (t *tar) Publish(_ context.Context, br build.Result, s string) (name.Reference, error) {
	s = strings.TrimPrefix(s, build.StrictScheme)
	// https://github.com/google/go-containerregistry/issues/212
	s = strings.ToLower(s)

	// There's no way to write an index to a tarball, so attempt to downcast it to an image.
	img, ok := br.(v1.Image)
	if !ok {
		return nil, fmt.Errorf("failed to interpret %s result as image: %v", s, br)
	}

	for _, tagName := range t.tags {
		tag, err := name.ParseReference(fmt.Sprintf("%s:%s", t.namer(t.base, s), tagName))
		if err != nil {
			return nil, err
		}
		t.refs[tag] = img
	}

	h, err := img.Digest()
	if err != nil {
		return nil, err
	}

	if len(t.tags) == 0 {
		ref, err := name.ParseReference(fmt.Sprintf("%s@%s", t.namer(t.base, s), h))
		if err != nil {
			return nil, err
		}
		t.refs[ref] = img
	}

	ref := fmt.Sprintf("%s@%s", t.namer(t.base, s), h)
	if len(t.tags) == 1 && t.tags[0] != latestTag {
		// If a single tag is explicitly set (not latest), then this
		// is probably a release, so include the tag in the reference.
		ref = fmt.Sprintf("%s:%s@%s", t.namer(t.base, s), t.tags[0], h)
	}
	dig, err := name.NewDigest(ref)
	if err != nil {
		return nil, err
	}

	return &dig, nil
}

func (t *tar) Close() error {
	log.Printf("Saving %v", t.file)
	if err := tarball.MultiRefWriteToFile(t.file, t.refs); err != nil {
		// Bad practice, but we log  this here because right now we just defer the Close.
		log.Printf("failed to save %q: %v", t.file, err)
		return err
	}
	log.Printf("Saved %v", t.file)
	return nil
}
