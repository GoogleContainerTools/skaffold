package cache

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
)

type ImageCache struct {
	docker *client.Client
	image  string
}

func NewImageCache(imageRef name.Reference, dockerClient *client.Client) *ImageCache {
	sum := sha256.Sum256([]byte(imageRef.Name()))
	return &ImageCache{
		image:  fmt.Sprintf("pack-cache-%x", sum[:6]),
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
