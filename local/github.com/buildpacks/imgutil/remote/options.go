package remote

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

// AddEmptyLayerOnSave adds an empty layer before saving if the image has no layers at all.
// This option is useful when exporting to registries that do not allow saving an image without layers,
// for example: gcr.io.
func AddEmptyLayerOnSave() func(*imgutil.ImageOptions) {
	return func(o *imgutil.ImageOptions) {
		o.AddEmptyLayerOnSave = true
	}
}

// WithRegistrySetting registers options to use when accessing images in a registry
// in order to construct the image.
// The referenced images could include the base image, a previous image, or the image itself.
// The insecure parameter allows image references to be fetched without TLS.
func WithRegistrySetting(repository string, insecure bool) func(*imgutil.ImageOptions) {
	return func(o *imgutil.ImageOptions) {
		if o.RegistrySettings == nil {
			o.RegistrySettings = make(map[string]imgutil.RegistrySetting)
		}
		o.RegistrySettings[repository] = imgutil.RegistrySetting{
			Insecure: insecure,
		}
	}
}

// FIXME: the following functions are defined in this package for backwards compatibility,
// and should eventually be deprecated.

func FromBaseImage(name string) func(*imgutil.ImageOptions) {
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
