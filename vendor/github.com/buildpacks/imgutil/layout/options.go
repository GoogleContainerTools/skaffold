package layout

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

type ImageOption func(*options) error

type options struct {
	platform      imgutil.Platform
	baseImage     v1.Image
	baseImagePath string
	prevImagePath string
	withHistory   bool
	createdAt     time.Time
	mediaTypes    imgutil.MediaTypes
}

// FromBaseImage loads the given image as the config and layers for the new image.
// Ignored if image is not found.
func FromBaseImage(base v1.Image) ImageOption {
	return func(i *options) error {
		i.baseImage = base
		return nil
	}
}

// FromBaseImagePath (layout only) loads an existing image as the config and layers for the new underlyingImage.
// Ignored if underlyingImage is not found.
func FromBaseImagePath(path string) ImageOption {
	return func(i *options) error {
		i.baseImagePath = path
		return nil
	}
}

// WithCreatedAt lets a caller set the created at timestamp for the image.
// Defaults for a new image is imgutil.NormalizedDateTime
func WithCreatedAt(createdAt time.Time) ImageOption {
	return func(i *options) error {
		i.createdAt = createdAt
		return nil
	}
}

// WithDefaultPlatform provides Architecture/OS/OSVersion defaults for the new image.
// Defaults for a new image are ignored when FromBaseImage returns an image.
// FromBaseImage and WithPreviousImage will use the platform to choose an image from a manifest list.
func WithDefaultPlatform(platform imgutil.Platform) ImageOption {
	return func(i *options) error {
		i.platform = platform
		return nil
	}
}

// WithHistory if provided will configure the image to preserve history when saved
// (including any history from the base image if valid).
func WithHistory() ImageOption {
	return func(opts *options) error {
		opts.withHistory = true
		return nil
	}
}

// WithMediaTypes lets a caller set the desired media types for the image manifest and config files,
// including the layers referenced in the manifest, to be either OCI media types or Docker media types.
func WithMediaTypes(requested imgutil.MediaTypes) ImageOption {
	return func(i *options) error {
		i.mediaTypes = requested
		return nil
	}
}

// WithPreviousImage loads an existing image as a source for reusable layers.
// Use with ReuseLayer().
// Ignored if underlyingImage is not found.
func WithPreviousImage(path string) ImageOption {
	return func(i *options) error {
		i.prevImagePath = path
		return nil
	}
}
