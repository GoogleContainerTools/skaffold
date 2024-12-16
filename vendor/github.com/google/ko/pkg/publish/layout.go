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

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/ko/pkg/build"
)

type LayoutPublisher struct {
	p string
}

// NewLayout returns a new publish.Interface that saves images to an OCI Image Layout.
func NewLayout(p string) Interface {
	return &LayoutPublisher{p}
}

func (l *LayoutPublisher) writeResult(br build.Result) (layout.Path, error) {
	p, err := layout.FromPath(l.p)
	if err != nil {
		p, err = layout.Write(l.p, empty.Index)
		if err != nil {
			return "", err
		}
	}

	mt, err := br.MediaType()
	if err != nil {
		return "", err
	}

	switch mt {
	case types.OCIImageIndex, types.DockerManifestList:
		idx, ok := br.(v1.ImageIndex)
		if !ok {
			return "", fmt.Errorf("failed to interpret result as index: %v", br)
		}
		if err := p.AppendIndex(idx); err != nil {
			return "", err
		}
		return p, nil
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		img, ok := br.(v1.Image)
		if !ok {
			return "", fmt.Errorf("failed to interpret result as image: %v", br)
		}
		if err := p.AppendImage(img); err != nil {
			return "", err
		}
		return p, nil
	default:
		return "", fmt.Errorf("result image media type: %s", mt)
	}
}

// Publish implements publish.Interface.
func (l *LayoutPublisher) Publish(_ context.Context, br build.Result, s string) (name.Reference, error) {
	log.Printf("Saving %v", s)
	p, err := l.writeResult(br)
	if err != nil {
		return nil, err
	}
	log.Printf("Saved %v", s)

	h, err := br.Digest()
	if err != nil {
		return nil, err
	}

	dig, err := name.NewDigest(fmt.Sprintf("%s@%s", p, h))
	if err != nil {
		return nil, err
	}

	return dig, nil
}

func (l *LayoutPublisher) Close() error {
	return nil
}
