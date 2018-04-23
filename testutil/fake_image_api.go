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

package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/moby/moby/client"
	"github.com/moby/moby/pkg/jsonmessage"
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
	ErrImagePush      bool

	BuildImageID string

	ReturnBody io.ReadCloser
}

func NewFakeImageAPIClient(initContents map[string]string, opts *FakeImageAPIOptions) *FakeImageAPIClient {
	if opts == nil {
		opts = &FakeImageAPIOptions{}
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

	imageID := "sha256:imageid"
	f.tagToImageID[imageID] = imageID

	if f.opts.ReturnBody != nil {
		return types.ImageBuildResponse{
			Body: f.opts.ReturnBody,
		}, nil
	}

	aux := json.RawMessage([]byte(fmt.Sprintf(`{"ID":"%s"}`, imageID)))
	buf, _ := json.Marshal(jsonmessage.JSONMessage{
		Aux: &aux,
	})

	return types.ImageBuildResponse{
		Body: ioutil.NopCloser(bytes.NewReader(buf)),
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

func (f *FakeImageAPIClient) ImagePush(_ context.Context, _ string, _ types.ImagePushOptions) (io.ReadCloser, error) {
	var err error
	if f.opts.ErrImagePush {
		err = fmt.Errorf("")
	}

	if f.opts.ReturnBody == nil {
		f.opts.ReturnBody = FakeReaderCloser{Err: io.EOF}
	}

	return f.opts.ReturnBody, err
}

func (f *FakeImageAPIClient) Info(ctx context.Context) (types.Info, error) {
	return types.Info{
		IndexServerAddress: registry.IndexServer,
	}, nil
}

func (f *FakeImageAPIClient) Close() error { return nil }
