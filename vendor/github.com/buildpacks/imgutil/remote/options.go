package remote

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

type ImageOption func(*options) error

type options struct {
	platform            imgutil.Platform
	baseImageRepoName   string
	prevImageRepoName   string
	createdAt           time.Time
	addEmptyLayerOnSave bool
	withHistory         bool
	registrySettings    map[string]registrySetting
	mediaTypes          imgutil.MediaTypes
	config              *v1.Config
}

// AddEmptyLayerOnSave (remote only) adds an empty layer before saving if the image has no layer at all.
// This option is useful when exporting to registries that do not allow saving an image without layers,
// for example: gcr.io
func AddEmptyLayerOnSave() ImageOption {
	return func(opts *options) error {
		opts.addEmptyLayerOnSave = true
		return nil
	}
}

// FromBaseImage loads an existing image as the config and layers for the new image.
// Ignored if image is not found.
func FromBaseImage(imageName string) ImageOption {
	return func(opts *options) error {
		opts.baseImageRepoName = imageName
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

func WithConfig(config *v1.Config) ImageOption {
	return func(opts *options) error {
		opts.config = config
		return nil
	}
}

// WithDefaultPlatform provides Architecture/OS/OSVersion defaults for the new image.
// Defaults for a new image are ignored when FromBaseImage returns an image.
// FromBaseImage and WithPreviousImage will use the platform to choose an image from a manifest list.
func WithDefaultPlatform(platform imgutil.Platform) ImageOption {
	return func(opts *options) error {
		opts.platform = platform
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
// Ignored if image is not found.
func WithPreviousImage(imageName string) ImageOption {
	return func(opts *options) error {
		opts.prevImageRepoName = imageName
		return nil
	}
}

// WithRegistrySetting (remote only) registers options to use when accessing images in a registry in order to construct
// the image. The referenced images could include the base image, a previous image, or the image itself.
func WithRegistrySetting(repository string, insecure, insecureSkipVerify bool) ImageOption {
	return func(opts *options) error {
		opts.registrySettings[repository] = registrySetting{
			insecure:           insecure,
			insecureSkipVerify: insecureSkipVerify,
		}
		return nil
	}
}

// v1Options is used to configure the behavior when a v1.Image is created
type v1Options struct {
	platform        imgutil.Platform
	registrySetting registrySetting
}

type V1ImageOption func(*v1Options) error

// WithV1DefaultPlatform provides Architecture/OS/OSVersion defaults for the new v1.Image.
func WithV1DefaultPlatform(platform imgutil.Platform) V1ImageOption {
	return func(opts *v1Options) error {
		opts.platform = platform
		return nil
	}
}

// WithV1RegistrySetting registers options to use when accessing images in a registry in order to construct a v1.Image.
func WithV1RegistrySetting(insecure, insecureSkipVerify bool) V1ImageOption {
	return func(opts *v1Options) error {
		opts.registrySetting = registrySetting{
			insecure:           insecure,
			insecureSkipVerify: insecureSkipVerify,
		}
		return nil
	}
}
