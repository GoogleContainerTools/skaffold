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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	reg "github.com/docker/docker/registry"
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type ContainerState int

const (
	Created ContainerState = 0
	Started ContainerState = 1
)

type FakeAPIClient struct {
	client.CommonAPIClient

	tagToImageID    map[string]string
	ErrImageBuild   bool
	ErrImageInspect bool
	ErrImagePush    bool
	ErrImagePull    bool
	ErrStream       bool

	nextImageID int
	Pushed      map[string]string
	Built       []types.ImageBuildOptions

	nextContainerID int
	containerStates map[string]ContainerState
	containerLock   sync.Mutex
}

func (f *FakeAPIClient) Add(tag, imageID string) *FakeAPIClient {
	if f.tagToImageID == nil {
		f.tagToImageID = make(map[string]string)
	}

	f.tagToImageID[imageID] = imageID
	f.tagToImageID[tag] = imageID
	if !strings.Contains(tag, ":") {
		f.tagToImageID[tag+":latest"] = imageID
	}
	return f
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

	f.nextImageID++
	imageID := fmt.Sprintf("sha256:%d", f.nextImageID)

	for _, tag := range options.Tags {
		f.Add(tag, imageID)
	}

	f.Built = append(f.Built, options)

	return types.ImageBuildResponse{
		Body: f.body(imageID),
	}, nil
}

func (f *FakeAPIClient) ImageInspectWithRaw(_ context.Context, ref string) (types.ImageInspect, []byte, error) {
	if f.ErrImageInspect {
		return types.ImageInspect{}, nil, fmt.Errorf("")
	}

	for tag, imageID := range f.tagToImageID {
		if tag == ref || imageID == ref {
			rawConfig := []byte(fmt.Sprintf(`{"Config":{"Image":"%s"}}`, imageID))

			var repoDigests []string
			if digest, found := f.Pushed[ref]; found {
				repoDigests = append(repoDigests, ref+"@"+digest)
			}

			return types.ImageInspect{
				ID:          imageID,
				RepoDigests: repoDigests,
			}, rawConfig, nil
		}
	}

	return types.ImageInspect{}, nil, &notFoundError{}
}

func (f *FakeAPIClient) DistributionInspect(ctx context.Context, ref, encodedRegistryAuth string) (registry.DistributionInspect, error) {
	if sha, found := f.Pushed[ref]; found {
		return registry.DistributionInspect{
			Descriptor: v1.Descriptor{
				Digest: digest.Digest(sha),
			},
		}, nil
	}

	return registry.DistributionInspect{}, &notFoundError{}
}

func (f *FakeAPIClient) ImageTag(_ context.Context, image, ref string) error {
	imageID, ok := f.tagToImageID[image]
	if !ok {
		return fmt.Errorf("image %s not found", image)
	}

	f.Add(ref, imageID)
	return nil
}

func (f *FakeAPIClient) ImagePush(_ context.Context, ref string, _ types.ImagePushOptions) (io.ReadCloser, error) {
	if f.ErrImagePush {
		return nil, fmt.Errorf("")
	}

	sha256Digester := sha256.New()
	if _, err := sha256Digester.Write([]byte(f.tagToImageID[ref])); err != nil {
		return nil, err
	}

	digest := "sha256:" + fmt.Sprintf("%x", sha256Digester.Sum(nil))[0:64]
	if f.Pushed == nil {
		f.Pushed = make(map[string]string)
	}
	f.Pushed[ref] = digest

	return f.body(digest), nil
}

func (f *FakeAPIClient) ImagePull(context.Context, string, types.ImagePullOptions) (io.ReadCloser, error) {
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

	f.nextImageID++
	imageID := fmt.Sprintf("sha256:%d", f.nextImageID)
	f.Add(ref, imageID)

	return types.ImageLoadResponse{
		Body: f.body(imageID),
	}, nil
}

func (f *FakeAPIClient) ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *networktypes.NetworkingConfig, containerName string) (containertypes.ContainerCreateCreatedBody, error) {
	f.containerLock.Lock()
	defer f.containerLock.Unlock()

	f.nextContainerID++
	containerID := strconv.Itoa(f.nextContainerID)

	if f.containerStates == nil {
		f.containerStates = make(map[string]ContainerState)
	}
	f.containerStates[containerID] = Created

	return containertypes.ContainerCreateCreatedBody{
		ID: containerID,
	}, nil
}

func (f *FakeAPIClient) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	f.containerLock.Lock()
	defer f.containerLock.Unlock()

	state, present := f.containerStates[container]
	if !present {
		return errors.New("not found")
	}

	if state == Started {
		return errors.New("already started")
	}

	f.containerStates[container] = Started
	return nil
}

func (f *FakeAPIClient) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	f.containerLock.Lock()
	defer f.containerLock.Unlock()

	state, present := f.containerStates[container]
	if !present {
		return errors.New("not found")
	}

	if state != Started {
		return errors.New("not started")
	}

	return nil
}

func (f *FakeAPIClient) ContainerLogs(ctx context.Context, container string, config types.ContainerLogsOptions) (io.ReadCloser, error) {
	f.containerLock.Lock()
	defer f.containerLock.Unlock()

	state, present := f.containerStates[container]
	if !present {
		return nil, errors.New("not found")
	}

	if state != Started {
		return nil, errors.New("not started")
	}

	return ioutil.NopCloser(strings.NewReader("")), nil
}

func (f *FakeAPIClient) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
	f.containerLock.Lock()
	defer f.containerLock.Unlock()

	state, present := f.containerStates[container]
	if !present {
		return errors.New("not found")
	}

	if state != Started {
		return errors.New("not started")
	}

	delete(f.containerStates, container)
	return nil
}

func (f *FakeAPIClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return nil
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
