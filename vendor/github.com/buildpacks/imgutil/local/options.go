package local

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

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
