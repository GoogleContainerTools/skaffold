package fakes

import (
	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var _ imgutil.ImageIndex = ImageIndex{}

type ImageIndex struct {
	Manifests []v1.Descriptor
}

// AddManifest implements imgutil.ImageIndex.
func (i ImageIndex) AddManifest(_ v1.Image) {
	panic("unimplemented")
}

// Annotations implements imgutil.ImageIndex.
func (i ImageIndex) Annotations(_ name.Digest) (map[string]string, error) {
	panic("unimplemented")
}

// Architecture implements imgutil.ImageIndex.
func (i ImageIndex) Architecture(_ name.Digest) (string, error) {
	panic("unimplemented")
}

// DeleteDir implements imgutil.ImageIndex.
func (i ImageIndex) DeleteDir() error {
	panic("unimplemented")
}

// Inspect implements imgutil.ImageIndex.
func (i ImageIndex) Inspect() (string, error) {
	panic("unimplemented")
}

// OS implements imgutil.ImageIndex.
func (i ImageIndex) OS(_ name.Digest) (string, error) {
	panic("unimplemented")
}

// OSFeatures implements imgutil.ImageIndex.
func (i ImageIndex) OSFeatures(_ name.Digest) ([]string, error) {
	panic("unimplemented")
}

// OSVersion implements imgutil.ImageIndex.
func (i ImageIndex) OSVersion(_ name.Digest) (string, error) {
	panic("unimplemented")
}

// Push implements imgutil.ImageIndex.
func (i ImageIndex) Push(_ ...imgutil.IndexOption) error {
	panic("unimplemented")
}

// RemoveManifest implements imgutil.ImageIndex.
func (i ImageIndex) RemoveManifest(_ name.Digest) error {
	panic("unimplemented")
}

// SaveDir implements imgutil.ImageIndex.
func (i ImageIndex) SaveDir() error {
	panic("unimplemented")
}

// SetAnnotations implements imgutil.ImageIndex.
func (i ImageIndex) SetAnnotations(_ name.Digest, _ map[string]string) error {
	panic("unimplemented")
}

// SetArchitecture implements imgutil.ImageIndex.
func (i ImageIndex) SetArchitecture(_ name.Digest, _ string) error {
	panic("unimplemented")
}

// SetOS implements imgutil.ImageIndex.
func (i ImageIndex) SetOS(_ name.Digest, _ string) error {
	panic("unimplemented")
}

// SetVariant implements imgutil.ImageIndex.
func (i ImageIndex) SetVariant(_ name.Digest, _ string) error {
	panic("unimplemented")
}

// Variant implements imgutil.ImageIndex.
func (i ImageIndex) Variant(_ name.Digest) (string, error) {
	panic("unimplemented")
}

func (i ImageIndex) IndexManifest() (*v1.IndexManifest, error) {
	return &v1.IndexManifest{
		Manifests: i.Manifests,
	}, nil
}
