package layout

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

// FromBaseImageInstance loads the provided image as the manifest, config, and layers for the working image.
// If the image is not found, it does nothing.
func FromBaseImageInstance(image v1.Image) func(*imgutil.ImageOptions) {
	return func(o *imgutil.ImageOptions) {
		o.BaseImage = image
	}
}

// WithoutLayersWhenSaved (layout only) if provided will cause the image to be written without layers in the `blobs` directory.
func WithoutLayersWhenSaved() func(*imgutil.ImageOptions) {
	return func(o *imgutil.ImageOptions) {
		o.WithoutLayers = true
	}
}

// FIXME: the following functions are defined in this package for backwards compatibility,
// and should eventually be deprecated.

// FromBaseImagePath loads the image at the provided path as the manifest, config, and layers for the working image.
// If the image is not found, it does nothing.
func FromBaseImagePath(name string) func(*imgutil.ImageOptions) {
	return imgutil.FromBaseImage(name)
}

func WithConfig(c *v1.Config) func(*imgutil.ImageOptions) {
	return imgutil.WithConfig(c)
}

func WithCreatedAt(t time.Time) func(*imgutil.ImageOptions) {
	return imgutil.WithCreatedAt(t)
}

func WithDefaultPlatform(p imgutil.Platform) func(*imgutil.ImageOptions) {
	return imgutil.WithDefaultPlatform(p)
}

func WithHistory() func(*imgutil.ImageOptions) {
	return imgutil.WithHistory()
}

func WithMediaTypes(m imgutil.MediaTypes) func(*imgutil.ImageOptions) {
	return imgutil.WithMediaTypes(m)
}

func WithPreviousImage(name string) func(*imgutil.ImageOptions) {
	return imgutil.WithPreviousImage(name)
}
