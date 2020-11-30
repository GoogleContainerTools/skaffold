package pack

import (
	"context"

	"github.com/pkg/errors"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/config"
	"github.com/buildpacks/pack/internal/buildpack"
	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/style"
)

const (
	// Packaging indicator that format of inputs/outputs will be an OCI image on the registry.
	FormatImage = "image"

	// Packaging indicator that format of output will be a file on the host filesystem.
	FormatFile = "file"
)

// PackageBuildpackOptions is a configuration object used to define
// the behavior of PackageBuildpack.
type PackageBuildpackOptions struct {
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
	PullPolicy config.PullPolicy
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

	packageBuilder := buildpackage.NewBuilder(c.imageFactory)

	bpURI := opts.Config.Buildpack.URI
	if bpURI == "" {
		return errors.New("buildpack URI must be provided")
	}

	blob, err := c.downloader.Download(ctx, bpURI)
	if err != nil {
		return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(bpURI))
	}

	bp, err := dist.BuildpackFromRootBlob(blob, writerFactory)
	if err != nil {
		return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bpURI))
	}

	packageBuilder.SetBuildpack(bp)

	for _, dep := range opts.Config.Dependencies {
		var depBPs []dist.Buildpack

		if dep.URI != "" {
			if buildpack.HasDockerLocator(dep.URI) {
				imageName := buildpack.ParsePackageLocator(dep.URI)
				c.logger.Debugf("Downloading buildpack from image: %s", style.Symbol(imageName))
				mainBP, deps, err := extractPackagedBuildpacks(ctx, imageName, c.imageFetcher, opts.Publish, opts.PullPolicy)
				if err != nil {
					return err
				}

				depBPs = append([]dist.Buildpack{mainBP}, deps...)
			} else {
				blob, err := c.downloader.Download(ctx, dep.URI)
				if err != nil {
					return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(dep.URI))
				}

				isOCILayout, err := buildpackage.IsOCILayoutBlob(blob)
				if err != nil {
					return errors.Wrap(err, "inspecting buildpack blob")
				}

				if isOCILayout {
					mainBP, deps, err := buildpackage.BuildpacksFromOCILayoutBlob(blob)
					if err != nil {
						return errors.Wrapf(err, "extracting buildpacks from %s", style.Symbol(dep.URI))
					}

					depBPs = append([]dist.Buildpack{mainBP}, deps...)
				} else {
					depBP, err := dist.BuildpackFromRootBlob(blob, writerFactory)
					if err != nil {
						return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(dep.URI))
					}
					depBPs = []dist.Buildpack{depBP}
				}
			}
		} else if dep.ImageName != "" {
			c.logger.Warn("The 'image' key is deprecated. Use 'uri=\"docker://...\"' instead.")
			mainBP, deps, err := extractPackagedBuildpacks(ctx, dep.ImageName, c.imageFetcher, opts.Publish, opts.PullPolicy)
			if err != nil {
				return err
			}

			depBPs = append([]dist.Buildpack{mainBP}, deps...)
		}

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
