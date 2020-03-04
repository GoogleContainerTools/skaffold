package pack

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

func extractPackagedBuildpacks(ctx context.Context, pkgImageRef string, fetcher ImageFetcher, publish, noPull bool) (mainBP dist.Buildpack, depBPs []dist.Buildpack, err error) {
	pkgImage, err := fetcher.Fetch(ctx, pkgImageRef, !publish, !noPull)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fetching image %s", style.Symbol(pkgImageRef))
	}

	md := &buildpackage.Metadata{}
	if found, err := dist.GetLabel(pkgImage, buildpackage.MetadataLabel, md); err != nil {
		return nil, nil, err
	} else if !found {
		return nil, nil, errors.Errorf(
			"could not find label %s on image %s",
			style.Symbol(buildpackage.MetadataLabel),
			style.Symbol(pkgImageRef),
		)
	}

	bpLayers := dist.BuildpackLayers{}
	ok, err := dist.GetLabel(pkgImage, dist.BuildpackLayersLabel, &bpLayers)
	if err != nil {
		return nil, nil, err
	}

	if !ok {
		return nil, nil, errors.Errorf(
			"could not find label %s on image %s",
			style.Symbol(dist.BuildpackLayersLabel),
			style.Symbol(pkgImageRef),
		)
	}

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

			if desc.Info == md.BuildpackInfo {
				mainBP = dist.BuildpackFromTarBlob(desc, b)
			} else {
				depBPs = append(depBPs, dist.BuildpackFromTarBlob(desc, b))
			}
		}
	}

	return mainBP, depBPs, nil
}

type openerBlob struct {
	opener func() (io.ReadCloser, error)
}

func (b *openerBlob) Open() (io.ReadCloser, error) {
	return b.opener()
}
