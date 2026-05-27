package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/chainguard-dev/kaniko/pkg/util/proc"
	"github.com/google/go-containerregistry/pkg/name"
	dockerClient "github.com/moby/moby/client"

	cerrdefs "github.com/containerd/errdefs"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/logging"
)

const EnvVolumeKey = "PACK_VOLUME_KEY"

type VolumeCache struct {
	docker DockerClient
	volume string
}

func NewVolumeCache(imageRef name.Reference, cacheType CacheInfo, suffix string, dockerClient DockerClient, logger logging.Logger) (*VolumeCache, error) {
	var volumeName string
	if cacheType.Source == "" {
		volumeKey, err := getVolumeKey(imageRef, logger)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256([]byte(imageRef.Name() + volumeKey))
		vol := paths.FilterReservedNames(fmt.Sprintf("%s-%x", sanitizedRef(imageRef), sum[:6]))
		volumeName = fmt.Sprintf("pack-cache-%s.%s", vol, suffix)
	} else {
		volumeName = paths.FilterReservedNames(cacheType.Source)
	}

	return &VolumeCache{
		volume: volumeName,
		docker: dockerClient,
	}, nil
}

func getVolumeKey(imageRef name.Reference, logger logging.Logger) (string, error) {
	var foundKey string

	// first, look for key in env

	foundKey = os.Getenv(EnvVolumeKey)
	if foundKey != "" {
		return foundKey, nil
	}

	// then, look for key in existing config

	volumeKeysPath, err := config.DefaultVolumeKeysPath()
	if err != nil {
		return "", err
	}
	cfg, err := config.ReadVolumeKeys(volumeKeysPath)
	if err != nil {
		return "", err
	}

	foundKey = cfg.VolumeKeys[imageRef.Name()]
	if foundKey != "" {
		return foundKey, nil
	}

	// finally, create new key and store it in config

	// if we're running in a container, we should log a warning
	// so that we don't always re-create the cache
	if RunningInContainer() {
		logger.Warnf("%s is unset; set this environment variable to a secret value to avoid creating a new volume cache on every build", EnvVolumeKey)
	}

	newKey := randString(20)
	if cfg.VolumeKeys == nil {
		cfg.VolumeKeys = make(map[string]string)
	}
	cfg.VolumeKeys[imageRef.Name()] = newKey
	if err = config.Write(cfg, volumeKeysPath); err != nil {
		return "", err
	}

	return newKey, nil
}

// Returns a string iwith lowercase a-z, of length n
func randString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = 'a' + (b[i] % 26)
	}
	return string(b)
}

func (c *VolumeCache) Name() string {
	return c.volume
}

func (c *VolumeCache) Clear(ctx context.Context) error {
	_, err := c.docker.VolumeRemove(ctx, c.Name(), dockerClient.VolumeRemoveOptions{Force: true})
	if err != nil && !cerrdefs.IsNotFound(err) {
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

var RunningInContainer = func() bool {
	return proc.GetContainerRuntime(0, 0) != proc.RuntimeNotFound
}
