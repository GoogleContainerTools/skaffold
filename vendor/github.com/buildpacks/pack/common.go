package pack

import (
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

func (c *Client) parseTagReference(imageName string) (name.Reference, error) {
	if imageName == "" {
		return nil, errors.New("image is a required parameter")
	}
	if _, err := name.ParseReference(imageName, name.WeakValidation); err != nil {
		return nil, err
	}
	ref, err := name.NewTag(imageName, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a tag reference", imageName)
	}

	return ref, nil
}

func (c *Client) resolveRunImage(runImage, imgRegistry, bldrRegistry string, stackInfo builder.StackMetadata, additionalMirrors map[string][]string, publish bool) string {
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
		stackInfo.RunImage.Image,
		stackInfo.RunImage.Mirrors,
		additionalMirrors[stackInfo.RunImage.Image],
	)

	switch {
	case runImageName == stackInfo.RunImage.Image:
		c.logger.Debugf("Selected run image %s", style.Symbol(runImageName))
	case contains(stackInfo.RunImage.Mirrors, runImageName):
		c.logger.Debugf("Selected run image mirror %s", style.Symbol(runImageName))
	default:
		c.logger.Debugf("Selected run image mirror %s from local config", style.Symbol(runImageName))
	}
	return runImageName
}

func (c *Client) getRegistry(logger logging.Logger, registryName string) (registry.Cache, error) {
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

func getBestRunMirror(registry string, runImage string, mirrors []string, preferredMirrors []string) string {
	runImageList := append(preferredMirrors, append([]string{runImage}, mirrors...)...)
	for _, img := range runImageList {
		ref, err := name.ParseReference(img, name.WeakValidation)
		if err != nil {
			continue
		}
		if ref.Context().RegistryStr() == registry {
			return img
		}
	}

	if len(preferredMirrors) > 0 {
		return preferredMirrors[0]
	}

	return runImage
}
