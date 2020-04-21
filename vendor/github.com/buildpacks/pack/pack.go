package pack

import (
	"context"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

func extractPackagedBuildpacks(ctx context.Context, pkgImageRef string, fetcher ImageFetcher, publish, noPull bool) (mainBP dist.Buildpack, depBPs []dist.Buildpack, err error) {
	pkgImage, err := fetcher.Fetch(ctx, pkgImageRef, !publish, !noPull)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fetching image")
	}

	mainBP, depBPs, err = buildpackage.ExtractBuildpacks(pkgImage)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "extracting buildpacks from %s", style.Symbol(pkgImageRef))
	}

	return mainBP, depBPs, nil
}
