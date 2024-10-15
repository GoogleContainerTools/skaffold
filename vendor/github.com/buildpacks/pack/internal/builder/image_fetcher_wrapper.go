package builder

import (
	"context"

	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/pack/pkg/image"
)

type ImageFetcher interface {
	// Fetch fetches an image by resolving it both remotely and locally depending on provided parameters.
	// If daemon is true, it will look return a `local.Image`. Pull, applicable only when daemon is true, will
	// attempt to pull a remote image first.
	Fetch(ctx context.Context, name string, options image.FetchOptions) (imgutil.Image, error)

	// CheckReadAccess verifies if an image is accessible with read permissions
	// When FetchOptions.Daemon is true and the image doesn't exist in the daemon,
	// the behavior is dictated by the pull policy, which can have the following behavior
	//   - PullNever: returns false
	//   - PullAlways Or PullIfNotPresent: it will check read access for the remote image.
	// When FetchOptions.Daemon is false it will check read access for the remote image.
	CheckReadAccess(repo string, options image.FetchOptions) bool
}

type ImageFetcherWrapper struct {
	fetcher ImageFetcher
}

func NewImageFetcherWrapper(fetcher ImageFetcher) *ImageFetcherWrapper {
	return &ImageFetcherWrapper{
		fetcher: fetcher,
	}
}

func (w *ImageFetcherWrapper) Fetch(
	ctx context.Context,
	name string,
	options image.FetchOptions,
) (Inspectable, error) {
	return w.fetcher.Fetch(ctx, name, options)
}

func (w *ImageFetcherWrapper) CheckReadAccessValidator(repo string, options image.FetchOptions) bool {
	return w.fetcher.CheckReadAccess(repo, options)
}
