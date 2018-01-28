/*
Copyright 2018 Google LLC

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

package test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/moby/moby/client"
)

type FakeImageAPIClient struct {
	*client.Client
	tagToImageID map[string]string

	opts *FakeImageAPIOptions
}

type FakeImageAPIOptions struct {
	ErrImageBuild     bool
	ErrImageList      bool
	ErrImageListEmpty bool
	ErrImageTag       bool

	BuildImageID string

	ReturnBody io.ReadCloser
}

func NewFakeImageAPIClient(initContents map[string]string, opts *FakeImageAPIOptions) *FakeImageAPIClient {
	if opts == nil {
		opts = &FakeImageAPIOptions{}
	}
	if opts.ReturnBody == nil {
		opts.ReturnBody = FakeReaderCloser{Err: io.EOF}
	}
	return &FakeImageAPIClient{
		tagToImageID: initContents,
		opts:         opts,
	}
}

func (f *FakeImageAPIClient) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if f.opts.ErrImageBuild {
		return types.ImageBuildResponse{}, fmt.Errorf("")
	}
	for _, tag := range options.Tags {
		if !strings.Contains(tag, ":") {
			tag = fmt.Sprintf("%s:latest", tag)
		}
		if f.opts.BuildImageID == "" {
			f.tagToImageID[tag] = "sha256:imageid"
		} else {
			f.tagToImageID[tag] = f.opts.BuildImageID
		}
	}
	return types.ImageBuildResponse{
		Body: f.opts.ReturnBody,
	}, nil
}

func (f *FakeImageAPIClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	if f.opts.ErrImageList {
		return nil, fmt.Errorf("")
	}
	if f.opts.ErrImageListEmpty {
		return []types.ImageSummary{}, nil
	}
	ret := []types.ImageSummary{}
	if options.Filters.Contains("reference") {
		refs := options.Filters.Get("reference")
		for _, ref := range refs {
			imageID, ok := f.tagToImageID[ref]
			if !ok {
				continue
			}
			ret = append(ret, types.ImageSummary{
				ID:       imageID,
				RepoTags: []string{ref},
			})
		}
	}
	return ret, nil
}

func (f *FakeImageAPIClient) ImageTag(ctx context.Context, image, ref string) error {
	if f.opts.ErrImageTag {
		return fmt.Errorf("")
	}
	imageID, ok := f.tagToImageID[image]
	if !ok {
		return fmt.Errorf("image %s not found. fake registry contents: %s", image, f.tagToImageID)
	}
	f.tagToImageID[ref] = imageID
	return nil
}

func NewFakeImageAPIClientCloser() (client.ImageAPIClient, io.Closer, error) {
	fakeAPI := NewFakeImageAPIClient(map[string]string{}, &FakeImageAPIOptions{})
	return fakeAPI, fakeAPI, nil
}

func NewFakeImageAPIClientCloserBuildError() (client.ImageAPIClient, io.Closer, error) {
	fakeAPI := NewFakeImageAPIClient(map[string]string{}, &FakeImageAPIOptions{
		ErrImageBuild: true,
	})
	return fakeAPI, fakeAPI, nil
}

func NewFakeImageAPIClientCloserTagError() (client.ImageAPIClient, io.Closer, error) {
	fakeAPI := NewFakeImageAPIClient(map[string]string{}, &FakeImageAPIOptions{
		ErrImageTag: true,
	})
	return fakeAPI, fakeAPI, nil
}

func NewFakeImageAPIClientCloserListError() (client.ImageAPIClient, io.Closer, error) {
	fakeAPI := NewFakeImageAPIClient(map[string]string{}, &FakeImageAPIOptions{
		ErrImageList: true,
	})
	return fakeAPI, fakeAPI, nil
}

func (f *FakeImageAPIClient) Close() error { return nil }
