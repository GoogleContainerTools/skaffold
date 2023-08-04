package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

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

	vol := paths.FilterReservedNames(fmt.Sprintf("%s-%x", sanitizedRef(imageRef), sum[:6]))
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

func (c *VolumeCache) Type() Type {
	return Volume
}

// note image names and volume names are validated using the same restrictions:
// see https://github.com/moby/moby/blob/f266f13965d5bfb1825afa181fe6c32f3a597fa3/daemon/names/names.go#L5
func sanitizedRef(ref name.Reference) string {
	result := strings.TrimPrefix(ref.Context().String(), ref.Context().RegistryStr()+"/")
	result = strings.ReplaceAll(result, "/", "_")
	return fmt.Sprintf("%s_%s", result, ref.Identifier())
}
