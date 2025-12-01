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
	"iter"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerState int

const (
	Created ContainerState = 0
	Started ContainerState = 1

	TestUtilization uint64 = 424242
)

type FakeAPIClient struct {
	client.APIClient

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
	Built []client.ImageBuildOptions
	// ref -> [id]
	LocalImages map[string][]string
}

func (f *FakeAPIClient) ServerVersion(ctx context.Context, opts client.ServerVersionOptions) (client.ServerVersionResult, error) {
	if f.ErrVersion {
		return client.ServerVersionResult{}, errors.New("docker not found")
	}
	return client.ServerVersionResult{}, nil
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

func (e notFoundError) NotFound() {}

type errReader struct{}

func (f errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("") }

func (f *FakeAPIClient) body(digest string) io.ReadCloser {
	if f.ErrStream {
		return io.NopCloser(&errReader{})
	}

	return io.NopCloser(strings.NewReader(fmt.Sprintf(`{"aux":{"digest":"%s"}}`, digest)))
}

func (f *FakeAPIClient) ImageBuild(_ context.Context, _ io.Reader, options client.ImageBuildOptions) (client.ImageBuildResult, error) {
	if f.ErrImageBuild {
		return client.ImageBuildResult{}, fmt.Errorf("")
	}

	next := atomic.AddInt32(&f.nextImageID, 1)
	imageID := fmt.Sprintf("sha256:%d", next)

	for _, tag := range options.Tags {
		f.Add(tag, imageID)
	}

	f.mux.Lock()
	f.Built = append(f.Built, options)
	f.mux.Unlock()

	return client.ImageBuildResult{
		Body: f.body(imageID),
	}, nil
}

func (f *FakeAPIClient) ImageRemove(_ context.Context, _ string, _ client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	if f.ErrImageRemove {
		return client.ImageRemoveResult{}, fmt.Errorf("test error")
	}
	return client.ImageRemoveResult{}, nil
}

func (fd *FakeAPIClient) ImageInspect(ctx context.Context, ref string, opts ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	if fd.ErrImageInspect {
		return client.ImageInspectResult{}, fmt.Errorf("")
	}

	ref, imageID, err := fd.findImageID(ref)
	if err != nil {
		return client.ImageInspectResult{}, err
	}

	var repoDigests []string
	if digest, found := fd.pushed.Load(ref); found {
		repoDigests = append(repoDigests, ref+"@"+digest.(string))
	}

	return client.ImageInspectResult{
		InspectResponse: image.InspectResponse{
			ID:          imageID,
			RepoDigests: repoDigests,
		},
	}, nil
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

func (f *FakeAPIClient) DistributionInspect(ctx context.Context, ref string, opts client.DistributionInspectOptions) (client.DistributionInspectResult, error) {
	if sha, found := f.pushed.Load(ref); found {
		return client.DistributionInspectResult{
			DistributionInspect: registry.DistributionInspect{
				Descriptor: v1.Descriptor{
					Digest: digest.Digest(sha.(string)),
				},
			},
		}, nil
	}

	return client.DistributionInspectResult{}, &notFoundError{}
}

func (f *FakeAPIClient) ImageTag(_ context.Context, opts client.ImageTagOptions) (client.ImageTagResult, error) {
	imageID, ok := f.tagToImageID.Load(opts.Source)
	if !ok {
		return client.ImageTagResult{}, fmt.Errorf("image %s not found", opts.Source)
	}

	f.Add(opts.Target, imageID.(string))
	return client.ImageTagResult{}, nil
}

func (f *FakeAPIClient) ImagePush(_ context.Context, ref string, _ client.ImagePushOptions) (client.ImagePushResponse, error) {
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
	return fakeImagePullPushResponse{
		ReadCloser: f.body(digest),
	}, nil
}

func (f *FakeAPIClient) ImagePull(_ context.Context, ref string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
	f.pulled.Store(ref, ref)
	if f.ErrImagePull {
		return nil, fmt.Errorf("")
	}

	return fakeImagePullPushResponse{
		ReadCloser: f.body(ref),
	}, nil
}

func (f *FakeAPIClient) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	return client.SystemInfoResult{
		Info: system.Info{
			IndexServerAddress: "https://index.docker.io/v1/",
		},
	}, nil
}

func (f *FakeAPIClient) ImageLoad(ctx context.Context, input io.Reader, _ ...client.ImageLoadOption) (client.ImageLoadResult, error) {
	ref, err := ReadRefFromFakeTar(input)
	if err != nil {
		return nil, fmt.Errorf("reading tar")
	}

	next := atomic.AddInt32(&f.nextImageID, 1)
	imageID := fmt.Sprintf("sha256:%d", next)
	f.Add(ref, imageID)

	return f.body(imageID), nil
}

func (f *FakeAPIClient) ImageList(ctx context.Context, ops client.ImageListOptions) (client.ImageListResult, error) {
	if f.ErrImageList {
		return client.ImageListResult{}, fmt.Errorf("test error")
	}
	ref := ""
	for k := range ops.Filters["reference"] {
		ref = k
		break
	}

	var rt []image.Summary
	for i, tag := range f.LocalImages[ref] {
		rt = append(rt, image.Summary{
			ID:      tag,
			Created: int64(i),
		})
	}
	return client.ImageListResult{
		Items: rt,
	}, nil
}

func (f *FakeAPIClient) ImageHistory(ctx context.Context, image string, _ ...client.ImageHistoryOption) (client.ImageHistoryResult, error) {
	return client.ImageHistoryResult{}, nil
}

func (f *FakeAPIClient) DiskUsage(ctx context.Context, ops client.DiskUsageOptions) (client.DiskUsageResult, error) {
	// if DUFails is positive faile first DUFails errors and then return ok
	// if negative, return ok first DUFails times and then fail the rest
	if f.DUFails > 0 {
		f.DUFails--
		return client.DiskUsageResult{}, fmt.Errorf("test error")
	}
	if f.DUFails < 0 {
		if f.DUFails == -1 {
			f.DUFails = math.MaxInt32 - 1
		}
		f.DUFails++
	}
	return client.DiskUsageResult{
		Containers: client.ContainersDiskUsage{
			TotalSize: int64(TestUtilization),
		},
	}, nil
}

func (f *FakeAPIClient) Close() error { return nil }

func CreateFakeImageTar(ref string, path string) error {
	image, err := random.Image(1024, 1)
	if err != nil {
		return fmt.Errorf("failed to create fake image %w", err)
	}
	reference, err := name.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("failed to parse reference %w", err)
	}
	return tarball.WriteToFile(path, reference, image)
}

func ReadRefFromFakeTar(input io.Reader) (string, error) {
	manifest, err := tarball.LoadManifest(func() (io.ReadCloser, error) {
		return io.NopCloser(input), nil
	})
	if err != nil {
		return "", fmt.Errorf("loading manifest %w", err)
	}

	return manifest[0].RepoTags[0], nil
}

type fakeImagePullPushResponse struct {
	io.ReadCloser
}

func (fakeImagePullPushResponse) JSONMessages(ctx context.Context) iter.Seq2[jsonstream.Message, error] {
	return nil
}

func (fakeImagePullPushResponse) Wait(ctx context.Context) error {
	return nil
}
