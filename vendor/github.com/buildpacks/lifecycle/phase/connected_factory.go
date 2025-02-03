package phase

import (
	"fmt"

	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/lifecycle/log"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/cache"
	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/platform"
)

// ConnectedFactory is used to construct lifecycle phases that require access to an image repository
// (registry, layout directory, or docker daemon) and/or a cache.
type ConnectedFactory struct {
	platformAPI     *api.Version
	apiVerifier     BuildpackAPIVerifier
	cacheHandler    CacheHandler
	configHandler   ConfigHandler
	imageHandler    image.Handler
	registryHandler image.RegistryHandler
}

// NewConnectedFactory constructs a new ConnectedFactory.
func NewConnectedFactory(
	platformAPI *api.Version,
	apiVerifier BuildpackAPIVerifier,
	cacheHandler CacheHandler,
	configHandler ConfigHandler,
	imageHandler image.Handler,
	registryHandler image.RegistryHandler,
) *ConnectedFactory {
	return &ConnectedFactory{
		platformAPI:     platformAPI,
		apiVerifier:     apiVerifier,
		cacheHandler:    cacheHandler,
		configHandler:   configHandler,
		imageHandler:    imageHandler,
		registryHandler: registryHandler,
	}
}

func (f *ConnectedFactory) ensureRegistryAccess(inputs platform.LifecycleInputs) error {
	var readImages, writeImages []string
	writeImages = append(writeImages, inputs.CacheImageRef)
	if f.imageHandler.Kind() == image.RemoteKind {
		readImages = append(readImages, inputs.PreviousImageRef, inputs.RunImageRef)
		writeImages = append(writeImages, inputs.OutputImageRef)
		writeImages = append(writeImages, inputs.AdditionalTags...)
	}
	if err := f.registryHandler.EnsureReadAccess(readImages...); err != nil {
		return fmt.Errorf("validating registry read access: %w", err)
	}
	if err := f.registryHandler.EnsureWriteAccess(writeImages...); err != nil {
		return fmt.Errorf("validating registry write access: %w", err)
	}
	return nil
}

func (f *ConnectedFactory) getPreviousImage(imageRef string, launchCacheDir string, logger log.Logger) (imgutil.Image, error) {
	if imageRef == "" {
		return nil, nil
	}
	previousImage, err := f.imageHandler.InitImage(imageRef)
	if err != nil {
		return nil, fmt.Errorf("getting previous image: %w", err)
	}
	if launchCacheDir == "" || f.imageHandler.Kind() != image.LocalKind {
		return previousImage, nil
	}
	volumeCache, err := cache.NewVolumeCache(launchCacheDir, logger)
	if err != nil {
		return nil, fmt.Errorf("creating launch cache: %w", err)
	}
	return cache.NewCachingImage(previousImage, volumeCache), nil
}

func (f *ConnectedFactory) getRunImage(imageRef string) (imgutil.Image, error) {
	if imageRef == "" {
		return nil, nil
	}
	runImage, err := f.imageHandler.InitImage(imageRef)
	if err != nil {
		return nil, fmt.Errorf("getting run image: %w", err)
	}
	return runImage, nil
}
