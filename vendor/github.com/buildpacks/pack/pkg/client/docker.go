package client

import (
	"context"
	"io"

	dockerClient "github.com/moby/moby/client"
)

// DockerClient is the subset of client.APIClient which required by this package
type DockerClient interface {
	ImageHistory(ctx context.Context, image string, opts ...dockerClient.ImageHistoryOption) (dockerClient.ImageHistoryResult, error)
	ImageInspect(ctx context.Context, image string, opts ...dockerClient.ImageInspectOption) (dockerClient.ImageInspectResult, error)
	ImageTag(ctx context.Context, options dockerClient.ImageTagOptions) (dockerClient.ImageTagResult, error)
	ImageLoad(ctx context.Context, input io.Reader, opts ...dockerClient.ImageLoadOption) (dockerClient.ImageLoadResult, error)
	ImageSave(ctx context.Context, images []string, opts ...dockerClient.ImageSaveOption) (dockerClient.ImageSaveResult, error)
	ImageRemove(ctx context.Context, image string, options dockerClient.ImageRemoveOptions) (dockerClient.ImageRemoveResult, error)
	ImagePull(ctx context.Context, ref string, options dockerClient.ImagePullOptions) (dockerClient.ImagePullResponse, error)
	Info(ctx context.Context, options dockerClient.InfoOptions) (dockerClient.SystemInfoResult, error)
	ServerVersion(ctx context.Context, options dockerClient.ServerVersionOptions) (dockerClient.ServerVersionResult, error)
	VolumeRemove(ctx context.Context, volumeID string, options dockerClient.VolumeRemoveOptions) (dockerClient.VolumeRemoveResult, error)
	ContainerCreate(ctx context.Context, options dockerClient.ContainerCreateOptions) (dockerClient.ContainerCreateResult, error)
	CopyFromContainer(ctx context.Context, containerID string, options dockerClient.CopyFromContainerOptions) (dockerClient.CopyFromContainerResult, error)
	ContainerInspect(ctx context.Context, containerID string, options dockerClient.ContainerInspectOptions) (dockerClient.ContainerInspectResult, error)
	ContainerRemove(ctx context.Context, container string, options dockerClient.ContainerRemoveOptions) (dockerClient.ContainerRemoveResult, error)
	CopyToContainer(ctx context.Context, container string, options dockerClient.CopyToContainerOptions) (dockerClient.CopyToContainerResult, error)
	ContainerWait(ctx context.Context, containerID string, options dockerClient.ContainerWaitOptions) dockerClient.ContainerWaitResult
	ContainerAttach(ctx context.Context, container string, options dockerClient.ContainerAttachOptions) (dockerClient.ContainerAttachResult, error)
	ContainerStart(ctx context.Context, container string, options dockerClient.ContainerStartOptions) (dockerClient.ContainerStartResult, error)
	NetworkCreate(ctx context.Context, name string, options dockerClient.NetworkCreateOptions) (dockerClient.NetworkCreateResult, error)
	NetworkRemove(ctx context.Context, networkID string, options dockerClient.NetworkRemoveOptions) (dockerClient.NetworkRemoveResult, error)
}
