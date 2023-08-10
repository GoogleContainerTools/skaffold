package client

import (
	"context"

	"github.com/pkg/errors"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/image"
)

const (
	// Packaging indicator that format of inputs/outputs will be an OCI image on the registry.
	FormatImage = "image"

	// Packaging indicator that format of output will be a file on the host filesystem.
	FormatFile = "file"

	// CNBExtension is the file extension for a cloud native buildpack tar archive
	CNBExtension = ".cnb"
)

// PackageBuildpackOptions is a configuration object used to define
// the behavior of PackageBuildpack.
type PackageBuildpackOptions struct {
	// The base director to resolve relative assest from
	RelativeBaseDir string

	// The name of the output buildpack artifact.
	Name string

	// Type of output format, The options are the either the const FormatImage, or FormatFile.
	Format string

	// Defines the Buildpacks configuration.
	Config pubbldpkg.Config

	// Push resulting builder image up to a registry
	// specified in the Name variable.
	Publish bool

	// Strategy for updating images before packaging.
	PullPolicy image.PullPolicy

	// Name of the buildpack registry. Used to
	// add buildpacks to a package.
	Registry string
}

// PackageBuildpack packages buildpack(s) into either an image or file.
func (c *Client) PackageBuildpack(ctx context.Context, opts PackageBuildpackOptions) error {
	if opts.Format == "" {
		opts.Format = FormatImage
	}

	if opts.Config.Platform.OS == "windows" && !c.experimental {
		return NewExperimentError("Windows buildpackage support is currently experimental.")
	}

	err := c.validateOSPlatform(ctx, opts.Config.Platform.OS, opts.Publish, opts.Format)
	if err != nil {
		return err
	}

	writerFactory, err := layer.NewWriterFactory(opts.Config.Platform.OS)
	if err != nil {
		return errors.Wrap(err, "creating layer writer factory")
	}

	packageBuilder := buildpack.NewBuilder(c.imageFactory)

	bpURI := opts.Config.Buildpack.URI
	if bpURI == "" {
		return errors.New("buildpack URI must be provided")
	}

	mainBlob, err := c.downloadBuildpackFromURI(ctx, bpURI, opts.RelativeBaseDir)
	if err != nil {
		return err
	}

	bp, err := buildpack.FromRootBlob(mainBlob, writerFactory)
	if err != nil {
		return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bpURI))
	}

	packageBuilder.SetBuildpack(bp)

	for _, dep := range opts.Config.Dependencies {
		var depBPs []buildpack.Buildpack
		mainBP, deps, err := c.buildpackDownloader.Download(ctx, dep.URI, buildpack.DownloadOptions{
			RegistryName:    opts.Registry,
			RelativeBaseDir: opts.RelativeBaseDir,
			ImageOS:         opts.Config.Platform.OS,
			ImageName:       dep.ImageName,
			Daemon:          !opts.Publish,
			PullPolicy:      opts.PullPolicy,
		})

		if err != nil {
			return errors.Wrapf(err, "packaging dependencies (uri=%s,image=%s)", style.Symbol(dep.URI), style.Symbol(dep.ImageName))
		}

		depBPs = append([]buildpack.Buildpack{mainBP}, deps...)
		for _, depBP := range depBPs {
			packageBuilder.AddDependency(depBP)
		}
	}

	switch opts.Format {
	case FormatFile:
		return packageBuilder.SaveAsFile(opts.Name, opts.Config.Platform.OS)
	case FormatImage:
		_, err = packageBuilder.SaveAsImage(opts.Name, opts.Publish, opts.Config.Platform.OS)
		return errors.Wrapf(err, "saving image")
	default:
		return errors.Errorf("unknown format: %s", style.Symbol(opts.Format))
	}
}

func (c *Client) downloadBuildpackFromURI(ctx context.Context, uri, relativeBaseDir string) (blob.Blob, error) {
	absPath, err := paths.FilePathToURI(uri, relativeBaseDir)
	if err != nil {
		return nil, errors.Wrapf(err, "making absolute: %s", style.Symbol(uri))
	}
	uri = absPath

	c.logger.Debugf("Downloading buildpack from URI: %s", style.Symbol(uri))
	blob, err := c.downloader.Download(ctx, uri)
	if err != nil {
		return nil, errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(uri))
	}

	return blob, nil
}

func (c *Client) validateOSPlatform(ctx context.Context, os string, publish bool, format string) error {
	if publish || format == FormatFile {
		return nil
	}

	info, err := c.docker.Info(ctx)
	if err != nil {
		return err
	}

	if info.OSType != os {
		return errors.Errorf("invalid %s specified: DOCKER_OS is %s", style.Symbol("platform.os"), style.Symbol(info.OSType))
	}

	return nil
}
