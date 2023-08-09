package buildpack

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...interface{})
	Info(msg string)
	Infof(fmt string, v ...interface{})
	Warn(msg string)
	Warnf(fmt string, v ...interface{})
	Error(msg string)
	Errorf(fmt string, v ...interface{})
}

type ImageFetcher interface {
	Fetch(ctx context.Context, name string, options image.FetchOptions) (imgutil.Image, error)
}

type Downloader interface {
	Download(ctx context.Context, pathOrURI string) (blob.Blob, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_registry_resolver.go github.com/buildpacks/pack/pkg/buildpack RegistryResolver

type RegistryResolver interface {
	Resolve(registryName, bpURI string) (string, error)
}

type buildpackDownloader struct {
	logger           Logger
	imageFetcher     ImageFetcher
	downloader       Downloader
	registryResolver RegistryResolver
}

func NewDownloader(logger Logger, imageFetcher ImageFetcher, downloader Downloader, registryResolver RegistryResolver) *buildpackDownloader { //nolint:golint,gosimple
	return &buildpackDownloader{
		logger:           logger,
		imageFetcher:     imageFetcher,
		downloader:       downloader,
		registryResolver: registryResolver,
	}
}

type DownloadOptions struct {
	// Buildpack registry name. Defines where all registry buildpacks will be pulled from.
	RegistryName string

	// The base directory to use to resolve relative assets
	RelativeBaseDir string

	// The OS of the builder image
	ImageOS string

	// Deprecated: the older alternative to buildpack URI
	ImageName string

	Daemon bool

	PullPolicy image.PullPolicy
}

func (c *buildpackDownloader) Download(ctx context.Context, buildpackURI string, opts DownloadOptions) (Buildpack, []Buildpack, error) {
	var err error
	var locatorType LocatorType
	if buildpackURI == "" && opts.ImageName != "" {
		c.logger.Warn("The 'image' key is deprecated. Use 'uri=\"docker://...\"' instead.")
		buildpackURI = opts.ImageName
		locatorType = PackageLocator
	} else {
		locatorType, err = GetLocatorType(buildpackURI, opts.RelativeBaseDir, []dist.BuildpackInfo{})
		if err != nil {
			return nil, nil, err
		}
	}

	var mainBP Buildpack
	var depBPs []Buildpack
	switch locatorType {
	case PackageLocator:
		imageName := ParsePackageLocator(buildpackURI)
		c.logger.Debugf("Downloading buildpack from image: %s", style.Symbol(imageName))
		mainBP, depBPs, err = extractPackagedBuildpacks(ctx, imageName, c.imageFetcher, image.FetchOptions{Daemon: opts.Daemon, PullPolicy: opts.PullPolicy})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from registry %s", style.Symbol(buildpackURI))
		}
	case RegistryLocator:
		c.logger.Debugf("Downloading buildpack from registry: %s", style.Symbol(buildpackURI))
		address, err := c.registryResolver.Resolve(opts.RegistryName, buildpackURI)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "locating in registry: %s", style.Symbol(buildpackURI))
		}

		mainBP, depBPs, err = extractPackagedBuildpacks(ctx, address, c.imageFetcher, image.FetchOptions{Daemon: opts.Daemon, PullPolicy: opts.PullPolicy})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from registry %s", style.Symbol(buildpackURI))
		}
	case URILocator:
		buildpackURI, err = paths.FilePathToURI(buildpackURI, opts.RelativeBaseDir)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "making absolute: %s", style.Symbol(buildpackURI))
		}

		c.logger.Debugf("Downloading buildpack from URI: %s", style.Symbol(buildpackURI))

		blob, err := c.downloader.Download(ctx, buildpackURI)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(buildpackURI))
		}

		mainBP, depBPs, err = decomposeBuildpack(blob, opts.ImageOS)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from %s", style.Symbol(buildpackURI))
		}
	default:
		return nil, nil, fmt.Errorf("error reading %s: invalid locator: %s", buildpackURI, locatorType)
	}
	return mainBP, depBPs, nil
}

// decomposeBuildpack decomposes a buildpack blob into the main builder (order buildpack) and it's dependencies buildpacks.
func decomposeBuildpack(blob blob.Blob, imageOS string) (mainBP Buildpack, depBPs []Buildpack, err error) {
	isOCILayout, err := IsOCILayoutBlob(blob)
	if err != nil {
		return mainBP, depBPs, errors.Wrap(err, "inspecting buildpack blob")
	}

	if isOCILayout {
		mainBP, depBPs, err = BuildpacksFromOCILayoutBlob(blob)
		if err != nil {
			return mainBP, depBPs, errors.Wrap(err, "extracting buildpacks")
		}
	} else {
		layerWriterFactory, err := layer.NewWriterFactory(imageOS)
		if err != nil {
			return mainBP, depBPs, errors.Wrapf(err, "get tar writer factory for OS %s", style.Symbol(imageOS))
		}

		mainBP, err = FromRootBlob(blob, layerWriterFactory)
		if err != nil {
			return mainBP, depBPs, errors.Wrap(err, "reading buildpack")
		}
	}

	return mainBP, depBPs, nil
}

func extractPackagedBuildpacks(ctx context.Context, pkgImageRef string, fetcher ImageFetcher, fetchOptions image.FetchOptions) (mainBP Buildpack, depBPs []Buildpack, err error) {
	pkgImage, err := fetcher.Fetch(ctx, pkgImageRef, fetchOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fetching image")
	}

	mainBP, depBPs, err = ExtractBuildpacks(pkgImage)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "extracting buildpacks from %s", style.Symbol(pkgImageRef))
	}

	return mainBP, depBPs, nil
}
