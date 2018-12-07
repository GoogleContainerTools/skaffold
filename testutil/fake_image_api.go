/*
Copyright 2018 The Skaffold Authors

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
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
)

type FakeImageAPIClient struct {
	*client.Client
	tagToImageID map[string]string

	opts *FakeImageAPIOptions
}

type FakeImageAPIOptions struct {
	ErrImageBuild   bool
	ErrImageInspect bool
	ErrImageTag     bool
	ErrImagePush    bool

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
		imageID := "sha256:" + randomID()

		f.tagToImageID[tag] = imageID
		f.tagToImageID[imageID] = imageID
		if !strings.Contains(tag, ":") {
			f.tagToImageID[tag+":latest"] = imageID
		}
	}

	return types.ImageBuildResponse{
		Body: f.opts.ReturnBody,
	}, nil
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (f *FakeImageAPIClient) ImageInspectWithRaw(ctx context.Context, ref string) (types.ImageInspect, []byte, error) {
	if f.opts.ErrImageInspect {
		return types.ImageInspect{}, nil, fmt.Errorf("")
	}

	imageID, ok := f.tagToImageID[ref]
	if !ok {
		return types.ImageInspect{}, nil, nil
	}

	return types.ImageInspect{ID: imageID}, nil, nil
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
	return f.opts.ReturnBody, err
}

func (f *FakeImageAPIClient) Info(ctx context.Context) (types.Info, error) {
	return types.Info{
		IndexServerAddress: registry.IndexServer,
	}, nil
}

func (f *FakeImageAPIClient) Close() error { return nil }
