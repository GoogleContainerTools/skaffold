package build

import (
	"context"

	dockerClient "github.com/moby/moby/client"
)

type DockerClient interface {
	ImageRemove(ctx context.Context, image string, options dockerClient.ImageRemoveOptions) (dockerClient.ImageRemoveResult, error)
	VolumeRemove(ctx context.Context, volumeID string, options dockerClient.VolumeRemoveOptions) (dockerClient.VolumeRemoveResult, error)
	ContainerWait(ctx context.Context, containerID string, options dockerClient.ContainerWaitOptions) dockerClient.ContainerWaitResult
	ContainerAttach(ctx context.Context, container string, options dockerClient.ContainerAttachOptions) (dockerClient.ContainerAttachResult, error)
	ContainerStart(ctx context.Context, container string, options dockerClient.ContainerStartOptions) (dockerClient.ContainerStartResult, error)
	ContainerCreate(ctx context.Context, options dockerClient.ContainerCreateOptions) (dockerClient.ContainerCreateResult, error)
	CopyFromContainer(ctx context.Context, containerID string, options dockerClient.CopyFromContainerOptions) (dockerClient.CopyFromContainerResult, error)
	ContainerInspect(ctx context.Context, containerID string, options dockerClient.ContainerInspectOptions) (dockerClient.ContainerInspectResult, error)
	ContainerRemove(ctx context.Context, container string, options dockerClient.ContainerRemoveOptions) (dockerClient.ContainerRemoveResult, error)
	CopyToContainer(ctx context.Context, container string, options dockerClient.CopyToContainerOptions) (dockerClient.CopyToContainerResult, error)
	NetworkCreate(ctx context.Context, name string, options dockerClient.NetworkCreateOptions) (dockerClient.NetworkCreateResult, error)
	NetworkRemove(ctx context.Context, networkID string, options dockerClient.NetworkRemoveOptions) (dockerClient.NetworkRemoveResult, error)
}
