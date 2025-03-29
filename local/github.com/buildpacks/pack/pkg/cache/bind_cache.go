package cache

import (
	"context"
	"os"
)

type BindCache struct {
	docker DockerClient
	bind   string
}

func NewBindCache(cacheType CacheInfo, dockerClient DockerClient) *BindCache {
	return &BindCache{
		bind:   cacheType.Source,
		docker: dockerClient,
	}
}

func (c *BindCache) Name() string {
	return c.bind
}

func (c *BindCache) Clear(ctx context.Context) error {
	err := os.RemoveAll(c.bind)
	if err != nil {
		return err
	}
	return nil
}

func (c *BindCache) Type() Type {
	return Bind
}
