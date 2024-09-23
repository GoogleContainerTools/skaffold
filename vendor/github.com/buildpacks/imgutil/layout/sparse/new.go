package sparse

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
)

// NewImage returns a new Image saved on disk that can be modified
func NewImage(path string, from v1.Image, ops ...imgutil.ImageOption) (*layout.Image, error) {
	preserveDigest := func(opts *imgutil.ImageOptions) {
		opts.PreserveDigest = true
	}
	ops = append([]imgutil.ImageOption{
		layout.FromBaseImageInstance(from),
		layout.WithoutLayersWhenSaved(),
		preserveDigest,
	}, ops...)
	img, err := layout.NewImage(path, ops...)
	if err != nil {
		return nil, err
	}
	return img, nil
}
