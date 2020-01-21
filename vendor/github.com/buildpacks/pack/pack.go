package pack

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

func extractPackagedBuildpacks(ctx context.Context, pkgImageRef string, fetcher ImageFetcher, publish, noPull bool) ([]dist.Buildpack, error) {
	pkgImage, err := fetcher.Fetch(ctx, pkgImageRef, !publish, !noPull)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching image %s", style.Symbol(pkgImageRef))
	}

	bpLayers := dist.BuildpackLayers{}
	ok, err := dist.GetLabel(pkgImage, dist.BuildpackLayersLabel, &bpLayers)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, errors.Errorf(
			"label %s not present on package %s",
			style.Symbol(dist.BuildpackLayersLabel),
			style.Symbol(pkgImageRef),
		)
	}

	var bps []dist.Buildpack
	for bpID, v := range bpLayers {
		for bpVersion, bpInfo := range v {
			desc := dist.BuildpackDescriptor{
				API: bpInfo.API,
				Info: dist.BuildpackInfo{
					ID:      bpID,
					Version: bpVersion,
				},
				Stacks: bpInfo.Stacks,
				Order:  bpInfo.Order,
			}

			diffID := bpInfo.LayerDiffID // Allow use in closure
			b := &openerBlob{
				opener: func() (io.ReadCloser, error) {
					rc, err := pkgImage.GetLayer(diffID)
					if err != nil {
						return nil, errors.Wrapf(err,
							"extracting buildpack %s layer (diffID %s) from package %s",
							style.Symbol(desc.Info.FullName()),
							style.Symbol(diffID),
							style.Symbol(pkgImage.Name()),
						)
					}
					return rc, nil
				},
			}

			bps = append(bps, dist.BuildpackFromTarBlob(desc, b))
		}
	}
	return bps, nil
}

type openerBlob struct {
	opener func() (io.ReadCloser, error)
}

func (b *openerBlob) Open() (io.ReadCloser, error) {
	return b.opener()
}
