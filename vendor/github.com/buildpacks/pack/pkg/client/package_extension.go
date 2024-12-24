package client

import (
	"context"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

// PackageExtension packages extension(s) into either an image or file.
func (c *Client) PackageExtension(ctx context.Context, opts PackageBuildpackOptions) error {
	if opts.Format == "" {
		opts.Format = FormatImage
	}

	if opts.Config.Platform.OS == "windows" && !c.experimental {
		return NewExperimentError("Windows extensionpackage support is currently experimental.")
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

	exURI := opts.Config.Extension.URI
	if exURI == "" {
		return errors.New("extension URI must be provided")
	}

	mainBlob, err := c.downloadBuildpackFromURI(ctx, exURI, opts.RelativeBaseDir)
	if err != nil {
		return err
	}

	ex, err := buildpack.FromExtensionRootBlob(mainBlob, writerFactory, c.logger)
	if err != nil {
		return errors.Wrapf(err, "creating extension from %s", style.Symbol(exURI))
	}

	packageBuilder.SetExtension(ex)

	target := dist.Target{OS: opts.Config.Platform.OS}
	switch opts.Format {
	case FormatFile:
		return packageBuilder.SaveAsFile(opts.Name, target, map[string]string{})
	case FormatImage:
		_, err = packageBuilder.SaveAsImage(opts.Name, opts.Publish, target, map[string]string{})
		return errors.Wrapf(err, "saving image")
	default:
		return errors.Errorf("unknown format: %s", style.Symbol(opts.Format))
	}
}
