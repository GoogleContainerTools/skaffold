package pack

import (
	"context"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

type CreatePackageOptions struct {
	Name    string
	Config  buildpackage.Config
	Publish bool
	NoPull  bool
}

func (c *Client) CreatePackage(ctx context.Context, opts CreatePackageOptions) error {
	packageBuilder := buildpackage.NewBuilder(c.imageFactory)

	bpURI := opts.Config.Buildpack.URI
	if bpURI == "" {
		return errors.New("buildpack URI must be provided")
	}

	blob, err := c.downloader.Download(ctx, bpURI)
	if err != nil {
		return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(bpURI))
	}

	bp, err := dist.BuildpackFromRootBlob(blob)
	if err != nil {
		return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bpURI))
	}

	packageBuilder.SetBuildpack(bp)

	for _, dep := range opts.Config.Dependencies {
		if dep.URI != "" {
			blob, err := c.downloader.Download(ctx, dep.URI)
			if err != nil {
				return errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(dep.URI))
			}

			depBP, err := dist.BuildpackFromRootBlob(blob)
			if err != nil {
				return errors.Wrapf(err, "creating buildpack from %s", style.Symbol(dep.URI))
			}

			packageBuilder.AddDependency(depBP)
		} else if dep.ImageName != "" {
			bps, err := extractPackagedBuildpacks(ctx, dep.ImageName, c.imageFetcher, opts.Publish, opts.NoPull)
			if err != nil {
				return err
			}

			for _, depBP := range bps {
				packageBuilder.AddDependency(depBP)
			}
		}
	}

	_, err = packageBuilder.Save(opts.Name, opts.Publish)
	if err != nil {
		return errors.Wrapf(err, "saving image")
	}

	return err
}
