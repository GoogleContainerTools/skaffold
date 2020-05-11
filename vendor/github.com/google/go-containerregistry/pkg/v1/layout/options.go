package layout

import v1 "github.com/google/go-containerregistry/pkg/v1"

// Option is a functional option for Layout.
//
// TODO: We'll need to change this signature to support Sparse/Thin images.
// Or, alternatively, wrap it in a sparse.Image that returns an empty list for layers?
type Option func(*v1.Descriptor) error

// WithAnnotations adds annotations to the artifact descriptor.
func WithAnnotations(annotations map[string]string) Option {
	return func(desc *v1.Descriptor) error {
		if desc.Annotations == nil {
			desc.Annotations = make(map[string]string)
		}
		for k, v := range annotations {
			desc.Annotations[k] = v
		}

		return nil
	}
}

// WithURLs adds urls to the artifact descriptor.
func WithURLs(urls []string) Option {
	return func(desc *v1.Descriptor) error {
		if desc.URLs == nil {
			desc.URLs = []string{}
		}
		desc.URLs = append(desc.URLs, urls...)
		return nil
	}
}

// WithPlatform sets the platform of the artifact descriptor.
func WithPlatform(platform v1.Platform) Option {
	return func(desc *v1.Descriptor) error {
		desc.Platform = &platform
		return nil
	}
}
