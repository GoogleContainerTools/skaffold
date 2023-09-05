package local

import (
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/buildpacks/imgutil"
)

type ImageOption func(*options) error

type options struct {
	platform          imgutil.Platform
	baseImageRepoName string
	prevImageRepoName string
	withHistory       bool
	createdAt         time.Time
	config            *container.Config
}

// FromBaseImage loads an existing image as the config and layers for the new image.
// Ignored if image is not found.
func FromBaseImage(imageName string) ImageOption {
	return func(i *options) error {
		i.baseImageRepoName = imageName
		return nil
	}
}

// WithCreatedAt lets a caller set the created at timestamp for the image.
// Defaults for a new image is imgutil.NormalizedDateTime
func WithCreatedAt(createdAt time.Time) ImageOption {
	return func(opts *options) error {
		opts.createdAt = createdAt
		return nil
	}
}

func WithConfig(config *container.Config) ImageOption {
	return func(opts *options) error {
		opts.config = config
		return nil
	}
}

// WithDefaultPlatform provides Architecture/OS/OSVersion defaults for the new image.
// Defaults for a new image are ignored when FromBaseImage returns an image.
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

// WithPreviousImage loads an existing image as a source for reusable layers.
// Use with ReuseLayer().
// Ignored if image is not found.
func WithPreviousImage(imageName string) ImageOption {
	return func(i *options) error {
		i.prevImageRepoName = imageName
		return nil
	}
}
