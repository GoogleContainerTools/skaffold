package cache

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
)

type VolumeCache struct {
	docker *client.Client
	volume string
}

func NewVolumeCache(imageRef name.Reference, suffix string, dockerClient *client.Client) *VolumeCache {
	sum := sha256.Sum256([]byte(imageRef.Name()))
	return &VolumeCache{
		volume: fmt.Sprintf("pack-cache-%x.%s", sum[:6], suffix),
		docker: dockerClient,
	}
}

func (c *VolumeCache) Name() string {
	return c.volume
}

func (c *VolumeCache) Clear(ctx context.Context) error {
	err := c.docker.VolumeRemove(ctx, c.Name(), true)
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}
	return nil
}
