package imgutil

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type ImageOption func(*ImageOptions)

type ImageOptions struct {
	BaseImageRepoName     string
	PreviousImageRepoName string
	Config                *v1.Config
	CreatedAt             time.Time
	MediaTypes            MediaTypes
	Platform              Platform
	PreserveHistory       bool
	LayoutOptions
	RemoteOptions

	// These options must be specified in each implementation's image constructor
	BaseImage     v1.Image
	PreviousImage v1.Image
}

type LayoutOptions struct {
	PreserveDigest bool
	WithoutLayers  bool
}

type RemoteOptions struct {
	RegistrySettings    map[string]RegistrySetting
	AddEmptyLayerOnSave bool
}

type RegistrySetting struct {
	Insecure bool
}

// FromBaseImage loads the provided image as the manifest, config, and layers for the working image.
// If the image is not found, it does nothing.
func FromBaseImage(name string) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.BaseImageRepoName = name
	}
}

// WithConfig lets a caller provided a `config` object for the working image.
func WithConfig(c *v1.Config) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.Config = c
	}
}

// WithCreatedAt lets a caller set the "created at" timestamp for the working image when saved.
// If not provided, the default is NormalizedDateTime.
func WithCreatedAt(t time.Time) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.CreatedAt = t
	}
}

// WithDefaultPlatform provides the default Architecture/OS/OSVersion if no base image is provided,
// or if the provided image inputs (base and previous) are manifest lists.
func WithDefaultPlatform(p Platform) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.Platform = p
	}
}

// WithHistory if provided will configure the image to preserve history when saved
// (including any history from the base image if valid).
func WithHistory() func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.PreserveHistory = true
	}
}

// WithMediaTypes lets a caller set the desired media types for the manifest and config (including layers referenced in the manifest)
// to be either OCI media types or Docker media types.
func WithMediaTypes(m MediaTypes) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.MediaTypes = m
	}
}

// WithPreviousImage loads an existing image as the source for reusable layers.
// Use with ReuseLayer().
// If the image is not found, it does nothing.
func WithPreviousImage(name string) func(*ImageOptions) {
	return func(o *ImageOptions) {
		o.PreviousImageRepoName = name
	}
}

type IndexOption func(options *IndexOptions) error

type IndexOptions struct {
	BaseIndexRepoName string
	MediaType         types.MediaType
	LayoutIndexOptions
	RemoteIndexOptions
	IndexPushOptions

	// These options must be specified in each implementation's image index constructor
	BaseIndex v1.ImageIndex
}

type LayoutIndexOptions struct {
	XdgPath string
}

type RemoteIndexOptions struct {
	Keychain authn.Keychain
	Insecure bool
}

// FromBaseIndex sets the name to use when loading the index.
// It used to either construct the path (if using layout) or the repo name (if using remote).
// If the index is not found, it does nothing.
func FromBaseIndex(name string) func(*IndexOptions) error {
	return func(o *IndexOptions) error {
		o.BaseIndexRepoName = name
		return nil
	}
}

// FromBaseIndexInstance sets the provided image index as the working image index.
func FromBaseIndexInstance(index v1.ImageIndex) func(options *IndexOptions) error {
	return func(o *IndexOptions) error {
		o.BaseIndex = index
		return nil
	}
}

// WithMediaType specifies the media type for the image index.
func WithMediaType(mediaType types.MediaType) func(options *IndexOptions) error {
	return func(o *IndexOptions) error {
		if !mediaType.IsIndex() {
			return fmt.Errorf("unsupported media type encountered: '%s'", mediaType)
		}
		o.MediaType = mediaType
		return nil
	}
}

// WithXDGRuntimePath Saves the Index to the '`xdgPath`/manifests'
func WithXDGRuntimePath(xdgPath string) func(options *IndexOptions) error {
	return func(o *IndexOptions) error {
		o.XdgPath = xdgPath
		return nil
	}
}

// WithKeychain fetches Index from registry with keychain
func WithKeychain(keychain authn.Keychain) func(options *IndexOptions) error {
	return func(o *IndexOptions) error {
		o.Keychain = keychain
		return nil
	}
}

// WithInsecure if true pulls and pushes the image to an insecure registry.
func WithInsecure() func(options *IndexOptions) error {
	return func(o *IndexOptions) error {
		o.Insecure = true
		return nil
	}
}

type IndexPushOptions struct {
	Purge           bool
	DestinationTags []string
}

// WithPurge if true deletes the index from the local filesystem after pushing
func WithPurge(purge bool) func(options *IndexOptions) error {
	return func(a *IndexOptions) error {
		a.Purge = purge
		return nil
	}
}

// WithTags sets the destination tags for the index when pushed
func WithTags(tags ...string) func(options *IndexOptions) error {
	return func(a *IndexOptions) error {
		a.DestinationTags = tags
		return nil
	}
}

func GetTransport(insecure bool) http.RoundTripper {
	if insecure {
		return &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402
			},
		}
	}
	return http.DefaultTransport
}
