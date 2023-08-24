package sparse

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil/layout"
)

// NewImage returns a new Image saved on disk that can be modified
func NewImage(path string, from v1.Image, ops ...layout.ImageOption) (*Image, error) {
	allOps := append([]layout.ImageOption{layout.FromBaseImage(from)}, ops...)
	img, err := layout.NewImage(path, allOps...)
	if err != nil {
		return nil, err
	}

	image := &Image{
		Image: *img,
	}
	return image, nil
}
