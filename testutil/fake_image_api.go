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
	"io/ioutil"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
)

type FakeAPIClient struct {
	client.CommonAPIClient

	TagToImageID    map[string]string
	ErrImageBuild   bool
	ErrImageInspect bool
	ErrImageTag     bool
	ErrImagePush    bool
	ErrImagePull    bool
	ErrStream       bool
}

type errReader struct{}

func (f errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("") }

func (f *FakeAPIClient) body(digest string) io.ReadCloser {
	if f.ErrStream {
		return ioutil.NopCloser(&errReader{})
	}

	return ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`{"aux":{"digest":"%s"}}`, digest)))
}

func (f *FakeAPIClient) ImageBuild(_ context.Context, _ io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if f.ErrImageBuild {
		return types.ImageBuildResponse{}, fmt.Errorf("")
	}

	if f.TagToImageID == nil {
		f.TagToImageID = make(map[string]string)
	}

	imageID := "sha256:" + randomID()
	f.TagToImageID[imageID] = imageID

	for _, tag := range options.Tags {
		f.TagToImageID[tag] = imageID
		if !strings.Contains(tag, ":") {
			f.TagToImageID[tag+":latest"] = imageID
		}
	}

	return types.ImageBuildResponse{
		Body: f.body(imageID),
	}, nil
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (f *FakeAPIClient) ImageInspectWithRaw(_ context.Context, ref string) (types.ImageInspect, []byte, error) {
	if f.ErrImageInspect {
		return types.ImageInspect{}, nil, fmt.Errorf("")
	}

	return types.ImageInspect{
		ID: f.TagToImageID[ref],
	}, nil, nil
}

func (f *FakeAPIClient) ImageTag(_ context.Context, image, ref string) error {
	if f.ErrImageTag {
		return fmt.Errorf("")
	}

	imageID, ok := f.TagToImageID[image]
	if !ok {
		return fmt.Errorf("image %s not found. fake registry contents: %s", image, f.TagToImageID)
	}

	if f.TagToImageID == nil {
		f.TagToImageID = make(map[string]string)
	}
	f.TagToImageID[ref] = imageID

	return nil
}

func (f *FakeAPIClient) ImagePush(_ context.Context, ref string, _ types.ImagePushOptions) (io.ReadCloser, error) {
	if f.ErrImagePush {
		return nil, fmt.Errorf("")
	}

	digest := f.TagToImageID[ref]

	return f.body(digest), nil
}

func (f *FakeAPIClient) ImagePull(_ context.Context, ref string, _ types.ImagePullOptions) (io.ReadCloser, error) {
	if f.ErrImagePull {
		return nil, fmt.Errorf("")
	}

	return f.body(""), nil
}

func (f *FakeAPIClient) Info(context.Context) (types.Info, error) {
	return types.Info{
		IndexServerAddress: registry.IndexServer,
	}, nil
}

func (f *FakeAPIClient) Close() error { return nil }
