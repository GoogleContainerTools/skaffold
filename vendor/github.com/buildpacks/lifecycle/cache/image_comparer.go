package cache

import (
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"
)

//go:generate mockgen -package testmockcache -destination ../phase/testmock/cache/image_comparer.go github.com/buildpacks/lifecycle/cache ImageComparer

// ImageComparer provides a way to compare images
type ImageComparer interface {
	ImagesEq(orig imgutil.Image, new imgutil.Image) (bool, error)
}

// ImageComparerImpl implements the ImageComparer interface
type ImageComparerImpl struct{}

// NewImageComparer instantiate ImageComparerImpl
func NewImageComparer() *ImageComparerImpl {
	return &ImageComparerImpl{}
}

// ImagesEq checks if the origin and the new images are the same
func (c *ImageComparerImpl) ImagesEq(origImage imgutil.Image, newImage imgutil.Image) (bool, error) {
	origIdentifier, err := origImage.Identifier()
	if err != nil {
		return false, errors.Wrap(err, "getting identifier for original image")
	}

	newIdentifier, err := newImage.Identifier()
	if err != nil {
		return false, errors.Wrap(err, "getting identifier for new image")
	}

	if origIdentifier.String() == newIdentifier.String() {
		return true, nil
	}

	return false, nil
}
