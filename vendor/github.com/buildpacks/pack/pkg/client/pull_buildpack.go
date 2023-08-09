package client

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

// PullBuildpackOptions are options available for PullBuildpack
type PullBuildpackOptions struct {
	// URI of the buildpack to retrieve.
	URI string
	// RegistryName to search for buildpacks from.
	RegistryName string
	// RelativeBaseDir to resolve relative assests from.
	RelativeBaseDir string
}

// PullBuildpack pulls given buildpack to be stored locally
func (c *Client) PullBuildpack(ctx context.Context, opts PullBuildpackOptions) error {
	locatorType, err := buildpack.GetLocatorType(opts.URI, "", []dist.BuildpackInfo{})
	if err != nil {
		return err
	}

	switch locatorType {
	case buildpack.PackageLocator:
		imageName := buildpack.ParsePackageLocator(opts.URI)
		c.logger.Debugf("Pulling buildpack from image: %s", imageName)

		_, err = c.imageFetcher.Fetch(ctx, imageName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
		if err != nil {
			return errors.Wrapf(err, "fetching image %s", style.Symbol(opts.URI))
		}
	case buildpack.RegistryLocator:
		c.logger.Debugf("Pulling buildpack from registry: %s", style.Symbol(opts.URI))
		registryCache, err := getRegistry(c.logger, opts.RegistryName)

		if err != nil {
			return errors.Wrapf(err, "invalid registry '%s'", opts.RegistryName)
		}

		registryBp, err := registryCache.LocateBuildpack(opts.URI)
		if err != nil {
			return errors.Wrapf(err, "locating in registry %s", style.Symbol(opts.URI))
		}

		_, err = c.imageFetcher.Fetch(ctx, registryBp.Address, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
		if err != nil {
			return errors.Wrapf(err, "fetching image %s", style.Symbol(opts.URI))
		}
	case buildpack.InvalidLocator:
		return fmt.Errorf("invalid buildpack URI %s", style.Symbol(opts.URI))
	default:
		return fmt.Errorf("unsupported buildpack URI type: %s", style.Symbol(locatorType.String()))
	}

	return nil
}
