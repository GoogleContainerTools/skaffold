package client

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient is the subset of CommonAPIClient which required by this package
type DockerClient interface {
	ImageHistory(ctx context.Context, image string) ([]image.HistoryResponseItem, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageTag(ctx context.Context, image, ref string) error
	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error)
	ImageSave(ctx context.Context, images []string) (io.ReadCloser, error)
	ImageRemove(ctx context.Context, image string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error)
	Info(ctx context.Context) (system.Info, error)
	ServerVersion(ctx context.Context) (types.Version, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
	ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *networktypes.NetworkingConfig, platform *specs.Platform, containerName string) (containertypes.CreateResponse, error)
	CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)
	ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error)
	ContainerRemove(ctx context.Context, container string, options containertypes.RemoveOptions) error
	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error
	ContainerWait(ctx context.Context, container string, condition containertypes.WaitCondition) (<-chan containertypes.WaitResponse, <-chan error)
	ContainerAttach(ctx context.Context, container string, options containertypes.AttachOptions) (types.HijackedResponse, error)
	ContainerStart(ctx context.Context, container string, options containertypes.StartOptions) error
	NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	NetworkRemove(ctx context.Context, network string) error
}
