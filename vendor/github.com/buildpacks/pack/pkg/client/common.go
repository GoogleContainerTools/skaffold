package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
)

func (c *Client) addManifestToIndex(ctx context.Context, repoName string, index imgutil.ImageIndex) error {
	imageRef, err := name.ParseReference(repoName, name.WeakValidation)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid manifest reference: %s", style.Symbol(repoName), err)
	}

	imageToAdd, err := c.imageFetcher.Fetch(ctx, imageRef.Name(), image.FetchOptions{Daemon: false})
	if err != nil {
		return err
	}

	index.AddManifest(imageToAdd.UnderlyingImage())
	return nil
}

func (c *Client) parseTagReference(imageName string) (name.Reference, error) {
	if imageName == "" {
		return nil, errors.New("image is a required parameter")
	}
	if _, err := name.ParseReference(imageName, name.WeakValidation); err != nil {
		return nil, fmt.Errorf("'%s' is not a valid tag reference: %s", imageName, err)
	}
	ref, err := name.NewTag(imageName, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a tag reference", imageName)
	}

	return ref, nil
}

func (c *Client) resolveRunImage(runImage, imgRegistry, bldrRegistry string, runImageMetadata builder.RunImageMetadata, additionalMirrors map[string][]string, publish bool, options image.FetchOptions) string {
	if runImage != "" {
		c.logger.Debugf("Using provided run-image %s", style.Symbol(runImage))
		return runImage
	}

	preferredRegistry := bldrRegistry
	if publish || bldrRegistry == "" {
		preferredRegistry = imgRegistry
	}

	runImageName := getBestRunMirror(
		preferredRegistry,
		runImageMetadata.Image,
		runImageMetadata.Mirrors,
		additionalMirrors[runImageMetadata.Image],
		c.imageFetcher,
		options,
	)

	switch {
	case runImageName == runImageMetadata.Image:
		c.logger.Debugf("Selected run image %s", style.Symbol(runImageName))
	case contains(runImageMetadata.Mirrors, runImageName):
		c.logger.Debugf("Selected run image mirror %s", style.Symbol(runImageName))
	default:
		c.logger.Debugf("Selected run image mirror %s from local config", style.Symbol(runImageName))
	}
	return runImageName
}

func getRegistry(logger logging.Logger, registryName string) (registry.Cache, error) {
	home, err := config.PackHome()
	if err != nil {
		return registry.Cache{}, err
	}

	if err := config.MkdirAll(home); err != nil {
		return registry.Cache{}, err
	}

	cfg, err := getConfig()
	if err != nil {
		return registry.Cache{}, err
	}

	if registryName == "" {
		return registry.NewDefaultRegistryCache(logger, home)
	}

	for _, reg := range config.GetRegistries(cfg) {
		if reg.Name == registryName {
			return registry.NewRegistryCache(logger, home, reg.URL)
		}
	}

	return registry.Cache{}, fmt.Errorf("registry %s is not defined in your config file", style.Symbol(registryName))
}

func getConfig() (config.Config, error) {
	path, err := config.DefaultConfigPath()
	if err != nil {
		return config.Config{}, err
	}

	cfg, err := config.Read(path)
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func contains(slc []string, v string) bool {
	for _, s := range slc {
		if s == v {
			return true
		}
	}
	return false
}

func getBestRunMirror(registry string, runImage string, mirrors []string, preferredMirrors []string, fetcher ImageFetcher, options image.FetchOptions) string {
	runImageList := filterImageList(append(append(append([]string{}, preferredMirrors...), runImage), mirrors...), fetcher, options)
	for _, img := range runImageList {
		ref, err := name.ParseReference(img, name.WeakValidation)
		if err != nil {
			continue
		}
		if reg := ref.Context().RegistryStr(); reg == registry {
			return img
		}
	}

	if len(runImageList) > 0 {
		return runImageList[0]
	}

	return runImage
}

func filterImageList(imageList []string, fetcher ImageFetcher, options image.FetchOptions) []string {
	var accessibleImages []string

	for i, img := range imageList {
		if fetcher.CheckReadAccess(img, options) {
			accessibleImages = append(accessibleImages, imageList[i])
		}
	}

	return accessibleImages
}
