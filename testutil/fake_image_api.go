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

package testutil

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	reg "github.com/docker/docker/registry"
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerState int

const (
	Created ContainerState = 0
	Started ContainerState = 1

	TestUtilization uint64 = 424242
)

type FakeAPIClient struct {
	client.CommonAPIClient

	ErrImageBuild   bool
	ErrImageInspect bool
	ErrImagePush    bool
	ErrImagePull    bool
	ErrImageList    bool
	ErrImageRemove  bool

	ErrStream  bool
	ErrVersion bool
	// will return the "test error" error on first <DUFails> DiskUsage calls
	DUFails int

	nextImageID  int32
	tagToImageID sync.Map // map[string]string
	pushed       sync.Map // map[string]string
	pulled       sync.Map // map[string]string

	mux   sync.Mutex
	Built []types.ImageBuildOptions
	// ref -> [id]
	LocalImages map[string][]string
}

func (f *FakeAPIClient) ServerVersion(ctx context.Context) (types.Version, error) {
	if f.ErrVersion {
		return types.Version{}, errors.New("docker not found")
	}
	return types.Version{}, nil
}

func (f *FakeAPIClient) Add(tag, imageID string) *FakeAPIClient {
	f.tagToImageID.Store(imageID, imageID)
	f.tagToImageID.Store(tag, imageID)
	if !strings.Contains(tag, ":") {
		f.tagToImageID.Store(tag+":latest", imageID)
	}
	return f
}

func (f *FakeAPIClient) Pulled() []string {
	var p []string
	f.pulled.Range(func(ref, _ interface{}) bool {
		p = append(p, ref.(string))
		return true
	})
	return p
}

func (f *FakeAPIClient) Pushed() map[string]string {
	p := make(map[string]string)
	f.pushed.Range(func(ref, id interface{}) bool {
		p[ref.(string)] = id.(string)
		return true
	})
	if len(p) == 0 {
		return nil
	}
	return p
}

type notFoundError struct {
	error
}

func (e notFoundError) NotFound() bool {
	return true
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

	next := atomic.AddInt32(&f.nextImageID, 1)
	imageID := fmt.Sprintf("sha256:%d", next)

	for _, tag := range options.Tags {
		f.Add(tag, imageID)
	}

	f.mux.Lock()
	f.Built = append(f.Built, options)
	f.mux.Unlock()

	return types.ImageBuildResponse{
		Body: f.body(imageID),
	}, nil
}

func (f *FakeAPIClient) ImageRemove(_ context.Context, _ string, _ types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	if f.ErrImageRemove {
		return []types.ImageDeleteResponseItem{}, fmt.Errorf("test error")
	}
	return []types.ImageDeleteResponseItem{}, nil
}

func (f *FakeAPIClient) ImageInspectWithRaw(_ context.Context, refOrID string) (types.ImageInspect, []byte, error) {
	if f.ErrImageInspect {
		return types.ImageInspect{}, nil, fmt.Errorf("")
	}

	ref, imageID, err := f.findImageID(refOrID)
	if err != nil {
		return types.ImageInspect{}, nil, err
	}

	rawConfig := []byte(fmt.Sprintf(`{"Config":{"Image":"%s"}}`, imageID))

	var repoDigests []string
	if digest, found := f.pushed.Load(ref); found {
		repoDigests = append(repoDigests, ref+"@"+digest.(string))
	}

	return types.ImageInspect{
		ID:          imageID,
		RepoDigests: repoDigests,
	}, rawConfig, nil
}

func (f *FakeAPIClient) findImageID(refOrID string) (string, string, error) {
	if id, found := f.tagToImageID.Load(refOrID); found {
		return refOrID, id.(string), nil
	}
	var ref, id string
	f.tagToImageID.Range(func(r, i interface{}) bool {
		if r == refOrID || i == refOrID {
			ref = r.(string)
			id = i.(string)
			return false
		}
		return true
	})
	if ref == "" {
		return "", "", &notFoundError{}
	}
	return ref, id, nil
}

func (f *FakeAPIClient) DistributionInspect(ctx context.Context, ref, encodedRegistryAuth string) (registry.DistributionInspect, error) {
	if sha, found := f.pushed.Load(ref); found {
		return registry.DistributionInspect{
			Descriptor: v1.Descriptor{
				Digest: digest.Digest(sha.(string)),
			},
		}, nil
	}

	return registry.DistributionInspect{}, &notFoundError{}
}

func (f *FakeAPIClient) ImageTag(_ context.Context, image, ref string) error {
	imageID, ok := f.tagToImageID.Load(image)
	if !ok {
		return fmt.Errorf("image %s not found", image)
	}

	f.Add(ref, imageID.(string))
	return nil
}

func (f *FakeAPIClient) ImagePush(_ context.Context, ref string, _ types.ImagePushOptions) (io.ReadCloser, error) {
	if f.ErrImagePush {
		return nil, fmt.Errorf("")
	}

	// use the digest if previously pushed
	imageID, found := f.tagToImageID.Load(ref)
	if !found {
		imageID = ""
	}
	sha256Digester := sha256.New()
	if _, err := sha256Digester.Write([]byte(imageID.(string))); err != nil {
		return nil, err
	}

	digest := "sha256:" + fmt.Sprintf("%x", sha256Digester.Sum(nil))[0:64]

	f.pushed.Store(ref, digest)
	return f.body(digest), nil
}

func (f *FakeAPIClient) ImagePull(_ context.Context, ref string, _ types.ImagePullOptions) (io.ReadCloser, error) {
	f.pulled.Store(ref, ref)
	if f.ErrImagePull {
		return nil, fmt.Errorf("")
	}

	return f.body(""), nil
}

func (f *FakeAPIClient) Info(context.Context) (types.Info, error) {
	return types.Info{
		IndexServerAddress: reg.IndexServer,
	}, nil
}

func (f *FakeAPIClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	ref, err := ReadRefFromFakeTar(input)
	if err != nil {
		return types.ImageLoadResponse{}, fmt.Errorf("reading tar")
	}

	next := atomic.AddInt32(&f.nextImageID, 1)
	imageID := fmt.Sprintf("sha256:%d", next)
	f.Add(ref, imageID)

	return types.ImageLoadResponse{
		Body: f.body(imageID),
	}, nil
}

func (f *FakeAPIClient) ImageList(ctx context.Context, ops types.ImageListOptions) ([]types.ImageSummary, error) {
	if f.ErrImageList {
		return []types.ImageSummary{}, fmt.Errorf("test error")
	}
	var rt []types.ImageSummary
	ref := ops.Filters.Get("reference")[0]

	for i, tag := range f.LocalImages[ref] {
		rt = append(rt, types.ImageSummary{
			ID:      tag,
			Created: int64(i),
		})
	}
	return rt, nil
}

func (f *FakeAPIClient) DiskUsage(ctx context.Context) (types.DiskUsage, error) {
	// if DUFails is positive faile first DUFails errors and then return ok
	// if negative, return ok first DUFails times and then fail the rest
	if f.DUFails > 0 {
		f.DUFails--
		return types.DiskUsage{}, fmt.Errorf("test error")
	}
	if f.DUFails < 0 {
		if f.DUFails == -1 {
			f.DUFails = math.MaxInt32 - 1
		}
		f.DUFails++
	}
	return types.DiskUsage{
		LayersSize: int64(TestUtilization),
	}, nil
}

func (f *FakeAPIClient) Close() error { return nil }

// TODO(dgageot): create something that looks more like an actual tar file.
func CreateFakeImageTar(ref string, path string) error {
	return ioutil.WriteFile(path, []byte(ref), os.ModePerm)
}

func ReadRefFromFakeTar(input io.Reader) (string, error) {
	buf, err := ioutil.ReadAll(input)
	if err != nil {
		return "", fmt.Errorf("reading tar")
	}

	return string(buf), nil
}
