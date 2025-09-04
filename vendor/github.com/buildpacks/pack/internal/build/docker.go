package build

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

type DockerClient interface {
	ImageRemove(ctx context.Context, image string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
	ContainerWait(ctx context.Context, container string, condition containertypes.WaitCondition) (<-chan containertypes.WaitResponse, <-chan error)
	ContainerAttach(ctx context.Context, container string, options containertypes.AttachOptions) (types.HijackedResponse, error)
	ContainerStart(ctx context.Context, container string, options containertypes.StartOptions) error
	ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *networktypes.NetworkingConfig, platform *specs.Platform, containerName string) (containertypes.CreateResponse, error)
	CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, containertypes.PathStat, error)
	ContainerInspect(ctx context.Context, container string) (containertypes.InspectResponse, error)
	ContainerRemove(ctx context.Context, container string, options containertypes.RemoveOptions) error
	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options containertypes.CopyToContainerOptions) error
	NetworkCreate(ctx context.Context, name string, options networktypes.CreateOptions) (networktypes.CreateResponse, error)
	NetworkRemove(ctx context.Context, network string) error
}

var _ DockerClient = dockerClient.APIClient(nil)
