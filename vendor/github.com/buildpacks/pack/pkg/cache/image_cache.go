package cache

import (
	"context"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/google/go-containerregistry/pkg/name"
	dockerClient "github.com/moby/moby/client"
)

type ImageCache struct {
	docker DockerClient
	image  string
}

type DockerClient interface {
	ImageRemove(ctx context.Context, image string, options dockerClient.ImageRemoveOptions) (dockerClient.ImageRemoveResult, error)
	VolumeRemove(ctx context.Context, volumeID string, options dockerClient.VolumeRemoveOptions) (dockerClient.VolumeRemoveResult, error)
}

func NewImageCache(imageRef name.Reference, dockerClient DockerClient) *ImageCache {
	return &ImageCache{
		image:  imageRef.Name(),
		docker: dockerClient,
	}
}

func (c *ImageCache) Name() string {
	return c.image
}

func (c *ImageCache) Clear(ctx context.Context) error {
	_, err := c.docker.ImageRemove(ctx, c.Name(), dockerClient.ImageRemoveOptions{
		Force: true,
	})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *ImageCache) Type() Type {
	return Image
}
