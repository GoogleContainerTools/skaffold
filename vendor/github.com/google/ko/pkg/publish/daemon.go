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
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/ko/pkg/build"
)

const (
	// LocalDomain is a sentinel "registry" that represents side-loading images into the daemon.
	LocalDomain = "ko.local"
)

// demon is intentionally misspelled to avoid name collision (and drive Jon nuts).
// [Narrator: Jon wasn't the only one driven nuts.]
type demon struct {
	base   string
	client daemon.Client
	namer  Namer
	tags   []string
}

// DaemonOption is a functional option for NewDaemon.
type DaemonOption func(*demon) error

// WithLocalDomain is a functional option for overriding the domain used for images that are side-loaded into the daemon.
func WithLocalDomain(domain string) DaemonOption {
	return func(i *demon) error {
		if domain != "" {
			i.base = domain
		}
		return nil
	}
}

// WithDockerClient is a functional option for overriding the docker client.
func WithDockerClient(client daemon.Client) DaemonOption {
	return func(i *demon) error {
		if client != nil {
			i.client = client
		}
		return nil
	}
}

// NewDaemon returns a new publish.Interface that publishes images to a container daemon.
func NewDaemon(namer Namer, tags []string, opts ...DaemonOption) (Interface, error) {
	d := &demon{
		base:  LocalDomain,
		namer: namer,
		tags:  tags,
	}
	for _, option := range opts {
		if err := option(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d *demon) getOpts(ctx context.Context) []daemon.Option {
	return []daemon.Option{
		daemon.WithContext(ctx),
		daemon.WithClient(d.client),
	}
}

// Publish implements publish.Interface
func (d *demon) Publish(ctx context.Context, br build.Result, s string) (name.Reference, error) {
	s = strings.TrimPrefix(s, build.StrictScheme)
	// https://github.com/google/go-containerregistry/issues/212
	s = strings.ToLower(s)

	// There's no way to write an index to a kind, so attempt to downcast it to an image.
	var img v1.Image
	switch i := br.(type) {
	case v1.Image:
		img = i
	case v1.ImageIndex:
		im, err := i.IndexManifest()
		if err != nil {
			return nil, err
		}
		goos, goarch := os.Getenv("GOOS"), os.Getenv("GOARCH")
		if goos == "" {
			goos = "linux"
		}
		if goarch == "" {
			goarch = "amd64"
		}
		for _, manifest := range im.Manifests {
			if manifest.Platform == nil {
				continue
			}
			if manifest.Platform.OS != goos {
				continue
			}
			if manifest.Platform.Architecture != goarch {
				continue
			}
			img, err = i.Image(manifest.Digest)
			if err != nil {
				return nil, err
			}
			break
		}
		if img == nil {
			return nil, fmt.Errorf("failed to find %s/%s image in index for image: %v", goos, goarch, s)
		}
	default:
		return nil, fmt.Errorf("failed to interpret %s result as image: %v", s, br)
	}

	h, err := img.Digest()
	if err != nil {
		return nil, err
	}

	digestTag, err := name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), h.Hex))
	if err != nil {
		return nil, err
	}

	log.Printf("Loading %v", digestTag)
	if resp, err := daemon.Write(digestTag, img, d.getOpts(ctx)...); err != nil {
		log.Println("daemon.Write response: ", resp)
		return nil, err
	}
	log.Printf("Loaded %v", digestTag)

	for _, tagName := range d.tags {
		log.Printf("Adding tag %v", tagName)
		tag, err := name.NewTag(fmt.Sprintf("%s:%s", d.namer(d.base, s), tagName))
		if err != nil {
			return nil, err
		}

		if err := daemon.Tag(digestTag, tag, d.getOpts(ctx)...); err != nil {
			return nil, err
		}
		log.Printf("Added tag %v", tagName)
	}

	return &digestTag, nil
}

func (d *demon) Close() error {
	return nil
}
