package imgutil

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ImageIndex an Interface with list of Methods required for creation and manipulation of v1.IndexManifest
type ImageIndex interface {
	// getters

	Annotations(digest name.Digest) (annotations map[string]string, err error)
	Architecture(digest name.Digest) (arch string, err error)
	OS(digest name.Digest) (os string, err error)
	OSFeatures(digest name.Digest) (osFeatures []string, err error)
	OSVersion(digest name.Digest) (osVersion string, err error)
	Variant(digest name.Digest) (osVariant string, err error)

	// setters

	SetAnnotations(digest name.Digest, annotations map[string]string) (err error)
	SetArchitecture(digest name.Digest, arch string) (err error)
	SetOS(digest name.Digest, os string) (err error)
	SetVariant(digest name.Digest, osVariant string) (err error)

	// misc

	Inspect() (string, error)
	AddManifest(image v1.Image)
	RemoveManifest(digest name.Digest) error

	Push(ops ...IndexOption) error
	SaveDir() error
	DeleteDir() error
}
