// Package cache provides functionalities around the cache
package cache

import (
	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/lifecycle/log"
)

//go:generate mockgen -package testmockcache -destination ../phase/testmock/cache/image_deleter.go github.com/buildpacks/lifecycle/cache ImageDeleter

// ImageDeleter defines the methods available to delete and compare cached images
type ImageDeleter interface {
	DeleteOrigImageIfDifferentFromNewImage(origImage, newImage imgutil.Image)
	DeleteImage(image imgutil.Image)
}

// ImageDeleterImpl is a component to manage cache image deletion
type ImageDeleterImpl struct {
	logger          log.Logger
	deletionEnabled bool
	comparer        ImageComparer
}

// NewImageDeleter creates a new ImageDeleter implementation
func NewImageDeleter(comparer ImageComparer, logger log.Logger, deletionEnabled bool) *ImageDeleterImpl {
	return &ImageDeleterImpl{comparer: comparer, logger: logger, deletionEnabled: deletionEnabled}
}

// DeleteOrigImageIfDifferentFromNewImage compares the two images, and it tries to delete it if they are not the same
func (c *ImageDeleterImpl) DeleteOrigImageIfDifferentFromNewImage(origImage, newImage imgutil.Image) {
	if c.deletionEnabled {
		same, err := c.comparer.ImagesEq(origImage, newImage)
		if err != nil {
			c.logger.Warnf("Unable to compare the image: %v", err.Error())
		}

		if !same {
			c.DeleteImage(origImage)
		}
	}
}

// DeleteImage deletes an image
func (c *ImageDeleterImpl) DeleteImage(image imgutil.Image) {
	if c.deletionEnabled {
		if err := image.Delete(); err != nil {
			c.logger.Warnf("Unable to delete cache image: %v", err.Error())
		}
	}
}
