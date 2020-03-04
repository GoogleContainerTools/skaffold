package pack

import (
	"context"

	"github.com/pkg/errors"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

type PackageBuildpackOptions struct {
	Name    string
	Config  pubbldpkg.Config
	Publish bool
	NoPull  bool
}

func (c *Client) PackageBuildpack(ctx context.Context, opts PackageBuildpackOptions) error {
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
			mainBP, depBPs, err := extractPackagedBuildpacks(ctx, dep.ImageName, c.imageFetcher, opts.Publish, opts.NoPull)
			if err != nil {
				return err
			}

			for _, depBP := range append([]dist.Buildpack{mainBP}, depBPs...) {
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
