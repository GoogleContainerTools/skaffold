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

func NewDownloader(logger Logger, imageFetcher ImageFetcher, downloader Downloader, registryResolver RegistryResolver) *buildpackDownloader { //nolint:revive,gosimple
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

	// The kind of module to download (valid values: "buildpack", "extension"). Defaults to "buildpack".
	ModuleKind string

	Daemon bool

	PullPolicy image.PullPolicy
}

func (c *buildpackDownloader) Download(ctx context.Context, moduleURI string, opts DownloadOptions) (BuildModule, []BuildModule, error) {
	kind := KindBuildpack
	if opts.ModuleKind == KindExtension {
		kind = KindExtension
	}

	var err error
	var locatorType LocatorType
	if moduleURI == "" && opts.ImageName != "" {
		c.logger.Warn("The 'image' key is deprecated. Use 'uri=\"docker://...\"' instead.")
		moduleURI = opts.ImageName
		locatorType = PackageLocator
	} else {
		locatorType, err = GetLocatorType(moduleURI, opts.RelativeBaseDir, []dist.ModuleInfo{})
		if err != nil {
			return nil, nil, err
		}
	}
	var mainBP BuildModule
	var depBPs []BuildModule
	switch locatorType {
	case PackageLocator:
		imageName := ParsePackageLocator(moduleURI)
		c.logger.Debugf("Downloading %s from image: %s", kind, style.Symbol(imageName))
		mainBP, depBPs, err = extractPackaged(ctx, kind, imageName, c.imageFetcher, image.FetchOptions{Daemon: opts.Daemon, PullPolicy: opts.PullPolicy})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from registry %s", style.Symbol(moduleURI))
		}
	case RegistryLocator:
		c.logger.Debugf("Downloading %s from registry: %s", kind, style.Symbol(moduleURI))
		address, err := c.registryResolver.Resolve(opts.RegistryName, moduleURI)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "locating in registry: %s", style.Symbol(moduleURI))
		}

		mainBP, depBPs, err = extractPackaged(ctx, kind, address, c.imageFetcher, image.FetchOptions{Daemon: opts.Daemon, PullPolicy: opts.PullPolicy})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from registry %s", style.Symbol(moduleURI))
		}
	case URILocator:
		moduleURI, err = paths.FilePathToURI(moduleURI, opts.RelativeBaseDir)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "making absolute: %s", style.Symbol(moduleURI))
		}

		c.logger.Debugf("Downloading %s from URI: %s", kind, style.Symbol(moduleURI))

		blob, err := c.downloader.Download(ctx, moduleURI)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "downloading %s from %s", kind, style.Symbol(moduleURI))
		}

		mainBP, depBPs, err = decomposeBlob(blob, kind, opts.ImageOS)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "extracting from %s", style.Symbol(moduleURI))
		}
	default:
		return nil, nil, fmt.Errorf("error reading %s: invalid locator: %s", moduleURI, locatorType)
	}
	return mainBP, depBPs, nil
}

// decomposeBlob decomposes a buildpack or extension blob into the main module (order buildpack or extension) and
// (for buildpack blobs) its dependent buildpacks.
func decomposeBlob(blob blob.Blob, kind string, imageOS string) (mainModule BuildModule, depModules []BuildModule, err error) {
	isOCILayout, err := IsOCILayoutBlob(blob)
	if err != nil {
		return mainModule, depModules, errors.Wrapf(err, "inspecting %s blob", kind)
	}

	if isOCILayout {
		mainModule, depModules, err = fromOCILayoutBlob(blob, kind)
		if err != nil {
			return mainModule, depModules, errors.Wrapf(err, "extracting %ss", kind)
		}
	} else {
		layerWriterFactory, err := layer.NewWriterFactory(imageOS)
		if err != nil {
			return mainModule, depModules, errors.Wrapf(err, "get tar writer factory for OS %s", style.Symbol(imageOS))
		}

		if kind == KindExtension {
			mainModule, err = FromExtensionRootBlob(blob, layerWriterFactory)
		} else {
			mainModule, err = FromBuildpackRootBlob(blob, layerWriterFactory)
		}
		if err != nil {
			return mainModule, depModules, errors.Wrapf(err, "reading %s", kind)
		}
	}

	return mainModule, depModules, nil
}

func fromOCILayoutBlob(blob blob.Blob, kind string) (mainModule BuildModule, depModules []BuildModule, err error) {
	switch kind {
	case KindBuildpack:
		mainModule, depModules, err = BuildpacksFromOCILayoutBlob(blob)
	case KindExtension:
		mainModule, err = ExtensionsFromOCILayoutBlob(blob)
	default:
		return nil, nil, fmt.Errorf("unknown module kind: %s", kind)
	}
	if err != nil {
		return nil, nil, err
	}
	return mainModule, depModules, nil
}

func extractPackaged(ctx context.Context, kind string, pkgImageRef string, fetcher ImageFetcher, fetchOptions image.FetchOptions) (mainModule BuildModule, depModules []BuildModule, err error) {
	pkgImage, err := fetcher.Fetch(ctx, pkgImageRef, fetchOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fetching image")
	}

	switch kind {
	case KindBuildpack:
		mainModule, depModules, err = extractBuildpacks(pkgImage)
	case KindExtension:
		mainModule, err = extractExtensions(pkgImage)
	default:
		return nil, nil, fmt.Errorf("unknown module kind: %s", kind)
	}
	if err != nil {
		return nil, nil, errors.Wrapf(err, "extracting %ss from %s", kind, style.Symbol(pkgImageRef))
	}
	return mainModule, depModules, nil
}
