package cache

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/paths"
)

type VolumeCache struct {
	docker client.CommonAPIClient
	volume string
}

func NewVolumeCache(imageRef name.Reference, suffix string, dockerClient client.CommonAPIClient) *VolumeCache {
	sum := sha256.Sum256([]byte(imageRef.Name()))

	vol := paths.FilterReservedNames(fmt.Sprintf("%x", sum[:6]))
	return &VolumeCache{
		volume: fmt.Sprintf("pack-cache-%s.%s", vol, suffix),
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
