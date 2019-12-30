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

package docker

import (
	"context"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/docker/docker/api/types"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// For testing
var (
	NewDockerAPI = NewDockerAPIImpl
)

type DockerAPI interface {
	// Remote Operations
	AddRemoteTag(src, target string) error
	RetrieveRemoteConfig(identifier string) (*v1.ConfigFile, error)
	PushTar(tarPath, tag string) (string, error)

	// Local Operations
	Close() error
	ExtraEnv() ([]string, error)
	ServerVersion(ctx context.Context) (types.Version, error)
	ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error)
	Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error)
	Push(ctx context.Context, out io.Writer, ref string) (string, error)
	Pull(ctx context.Context, out io.Writer, ref string) error
	Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error)
	Tag(ctx context.Context, image, ref string) error
	TagWithImageID(ctx context.Context, ref string, imageID string) (string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageExists(ctx context.Context, ref string) (bool, error)
	Prune(ctx context.Context, out io.Writer, images []string, pruneChildren bool) error
	ContainerRun(ctx context.Context, out io.Writer, runs ...ContainerRun) error
	CopyToContainer(ctx context.Context, container string, dest string, root string, paths []string, uid, gid int, modTime time.Time) error
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
}

type dockerAPI struct {
	runCtx *runcontext.RunContext
}

func NewDockerAPIImpl(runCtx *runcontext.RunContext) DockerAPI {
	return &dockerAPI{
		runCtx: runCtx,
	}
}

// Remote Operations

func (d *dockerAPI) AddRemoteTag(src, target string) error {
	return AddRemoteTag(src, target, d.runCtx.InsecureRegistries)
}

func (d *dockerAPI) RetrieveRemoteConfig(identifier string) (*v1.ConfigFile, error) {
	return RetrieveRemoteConfig(identifier, d.runCtx.InsecureRegistries)
}

func (d *dockerAPI) PushTar(tarPath, tag string) (string, error) {
	return Push(tarPath, tag, d.runCtx.InsecureRegistries)
}

// Local Operations

func (d *dockerAPI) Close() error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.Close()
}

func (d *dockerAPI) ExtraEnv() ([]string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return nil, err
	}

	return docker.ExtraEnv(), nil
}

func (d *dockerAPI) ServerVersion(ctx context.Context) (types.Version, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return types.Version{}, err
	}

	return docker.ServerVersion(ctx)
}

func (d *dockerAPI) ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return nil, err
	}

	return docker.ConfigFile(ctx, image)
}

func (d *dockerAPI) Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return "", err
	}

	return docker.Build(ctx, out, workspace, a, ref)
}

func (d *dockerAPI) Push(ctx context.Context, out io.Writer, ref string) (string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return "", err
	}

	return docker.Push(ctx, out, ref)
}

func (d *dockerAPI) Pull(ctx context.Context, out io.Writer, ref string) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.Pull(ctx, out, ref)
}

func (d *dockerAPI) Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return "", err
	}

	return docker.Load(ctx, out, input, ref)
}

func (d *dockerAPI) Tag(ctx context.Context, image, ref string) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.Tag(ctx, image, ref)
}

func (d *dockerAPI) TagWithImageID(ctx context.Context, ref string, imageID string) (string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return "", err
	}

	return docker.TagWithImageID(ctx, ref, imageID)
}

func (d *dockerAPI) ImageID(ctx context.Context, ref string) (string, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return "", err
	}

	return docker.ImageID(ctx, ref)
}

func (d *dockerAPI) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return types.ImageInspect{}, nil, err
	}

	return docker.ImageInspectWithRaw(ctx, image)
}

func (d *dockerAPI) ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return nil, err
	}

	return docker.ImageRemove(ctx, image, opts)
}

func (d *dockerAPI) ImageExists(ctx context.Context, ref string) (bool, error) {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return false, err
	}

	return docker.ImageExists(ctx, ref), nil
}

func (d *dockerAPI) Prune(ctx context.Context, out io.Writer, images []string, pruneChildren bool) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.Prune(ctx, out, images, pruneChildren)
}

func (d *dockerAPI) ContainerRun(ctx context.Context, out io.Writer, runs ...ContainerRun) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.ContainerRun(ctx, out, runs...)
}

func (d *dockerAPI) CopyToContainer(ctx context.Context, container string, dest string, root string, paths []string, uid, gid int, modTime time.Time) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.CopyToContainer(ctx, container, dest, root, paths, uid, gid, modTime)
}

func (d *dockerAPI) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	docker, err := NewAPIClientImpl(d.runCtx)
	if err != nil {
		return err
	}

	return docker.VolumeRemove(ctx, volumeID, force)
}
