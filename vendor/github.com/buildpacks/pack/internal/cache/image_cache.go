package cache

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
)

type ImageCache struct {
	docker client.CommonAPIClient
	image  string
}

func NewImageCache(imageRef name.Reference, dockerClient client.CommonAPIClient) *ImageCache {
	return &ImageCache{
		image:  imageRef.Name(),
		docker: dockerClient,
	}
}

func (c *ImageCache) Name() string {
	return c.image
}

func (c *ImageCache) Clear(ctx context.Context) error {
	_, err := c.docker.ImageRemove(ctx, c.Name(), types.ImageRemoveOptions{
		Force: true,
	})
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}
	return nil
}

func (c *ImageCache) Type() Type {
	return Image
}
