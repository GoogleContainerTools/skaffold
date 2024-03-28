//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mutate

import (
	"errors"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/empty"
	"github.com/sigstore/cosign/v2/pkg/oci/signed"
)

// Appendable is our signed version of mutate.Appendable
type Appendable interface {
	oci.SignedEntity
	mutate.Appendable
}

// IndexAddendum is our signed version of mutate.IndexAddendum
type IndexAddendum struct {
	Add Appendable
	v1.Descriptor
}

// AppendManifests is a form of mutate.AppendManifests that produces an
// oci.SignedImageIndex.  The index itself will contain no signatures,
// but allows access to the contained signed entities.
func AppendManifests(base v1.ImageIndex, adds ...IndexAddendum) oci.SignedImageIndex {
	madds := make([]mutate.IndexAddendum, 0, len(adds))
	for _, add := range adds {
		madds = append(madds, mutate.IndexAddendum{
			Add:        add.Add,
			Descriptor: add.Descriptor,
		})
	}
	return &indexWrapper{
		v1Index:  mutate.AppendManifests(base, madds...),
		ogbase:   base,
		addendum: adds,
	}
}

// We alias ImageIndex so that we can inline it without the type
// name colliding with the name of a method it had to implement.
type v1Index v1.ImageIndex

type indexWrapper struct {
	v1Index
	ogbase   v1Index
	addendum []IndexAddendum
}

var _ oci.SignedImageIndex = (*indexWrapper)(nil)

// Signatures implements oci.SignedImageIndex
func (i *indexWrapper) Signatures() (oci.Signatures, error) {
	return empty.Signatures(), nil
}

// Attestations implements oci.SignedImageIndex
func (i *indexWrapper) Attestations() (oci.Signatures, error) {
	return empty.Signatures(), nil
}

// Attachment implements oci.SignedImage
func (*indexWrapper) Attachment(name string) (oci.File, error) { //nolint: revive
	return nil, errors.New("unimplemented")
}

// SignedImage implements oci.SignedImageIndex
func (i *indexWrapper) SignedImage(h v1.Hash) (oci.SignedImage, error) {
	for _, add := range i.addendum {
		si, ok := add.Add.(oci.SignedImage)
		if !ok {
			continue
		}
		if d, err := si.Digest(); err != nil {
			return nil, err
		} else if d == h {
			return si, nil
		}
	}

	if sb, ok := i.ogbase.(oci.SignedImageIndex); ok {
		return sb.SignedImage(h)
	}

	unsigned, err := i.Image(h)
	if err != nil {
		return nil, err
	}
	return signed.Image(unsigned), nil
}

// SignedImageIndex implements oci.SignedImageIndex
func (i *indexWrapper) SignedImageIndex(h v1.Hash) (oci.SignedImageIndex, error) {
	for _, add := range i.addendum {
		sii, ok := add.Add.(oci.SignedImageIndex)
		if !ok {
			continue
		}
		if d, err := sii.Digest(); err != nil {
			return nil, err
		} else if d == h {
			return sii, nil
		}
	}

	if sb, ok := i.ogbase.(oci.SignedImageIndex); ok {
		return sb.SignedImageIndex(h)
	}

	unsigned, err := i.ImageIndex(h)
	if err != nil {
		return nil, err
	}
	return signed.ImageIndex(unsigned), nil
}

// AttachSignatureToEntity attaches the provided signature to the provided entity.
func AttachSignatureToEntity(se oci.SignedEntity, sig oci.Signature, opts ...SignOption) (oci.SignedEntity, error) {
	switch obj := se.(type) {
	case oci.SignedImage:
		return AttachSignatureToImage(obj, sig, opts...)
	case oci.SignedImageIndex:
		return AttachSignatureToImageIndex(obj, sig, opts...)
	default:
		return AttachSignatureToUnknown(obj, sig, opts...)
	}
}

// AttachAttestationToEntity attaches the provided attestation to the provided entity.
func AttachAttestationToEntity(se oci.SignedEntity, att oci.Signature, opts ...SignOption) (oci.SignedEntity, error) {
	switch obj := se.(type) {
	case oci.SignedImage:
		return AttachAttestationToImage(obj, att, opts...)
	case oci.SignedImageIndex:
		return AttachAttestationToImageIndex(obj, att, opts...)
	default:
		return AttachAttestationToUnknown(obj, att, opts...)
	}
}

// AttachFileToEntity attaches the provided file to the provided entity.
func AttachFileToEntity(se oci.SignedEntity, name string, f oci.File, opts ...SignOption) (oci.SignedEntity, error) {
	switch obj := se.(type) {
	case oci.SignedImage:
		return AttachFileToImage(obj, name, f, opts...)
	case oci.SignedImageIndex:
		return AttachFileToImageIndex(obj, name, f, opts...)
	default:
		return AttachFileToUnknown(obj, name, f, opts...)
	}
}

// AttachSignatureToImage attaches the provided signature to the provided image.
func AttachSignatureToImage(si oci.SignedImage, sig oci.Signature, opts ...SignOption) (oci.SignedImage, error) {
	return &signedImage{
		SignedImage: si,
		sig:         sig,
		attachments: make(map[string]oci.File),
		so:          makeSignOpts(opts...),
	}, nil
}

// AttachAttestationToImage attaches the provided attestation to the provided image.
func AttachAttestationToImage(si oci.SignedImage, att oci.Signature, opts ...SignOption) (oci.SignedImage, error) {
	return &signedImage{
		SignedImage: si,
		att:         att,
		attachments: make(map[string]oci.File),
		so:          makeSignOpts(opts...),
	}, nil
}

// AttachFileToImage attaches the provided file to the provided image.
func AttachFileToImage(si oci.SignedImage, name string, f oci.File, opts ...SignOption) (oci.SignedImage, error) {
	return &signedImage{
		SignedImage: si,
		attachments: map[string]oci.File{
			name: f,
		},
		so: makeSignOpts(opts...),
	}, nil
}

type signedImage struct {
	oci.SignedImage
	sig         oci.Signature
	att         oci.Signature
	so          *signOpts
	attachments map[string]oci.File
}

// Signatures implements oci.SignedImage
func (si *signedImage) Signatures() (oci.Signatures, error) {
	return si.so.dedupeAndReplace(si.sig, si.SignedImage.Signatures)
}

// Attestations implements oci.SignedImage
func (si *signedImage) Attestations() (oci.Signatures, error) {
	return si.so.dedupeAndReplace(si.att, si.SignedImage.Attestations)
}

// Attachment implements oci.SignedImage
func (si *signedImage) Attachment(attName string) (oci.File, error) {
	if f, ok := si.attachments[attName]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("attachment %q not found", attName)
}

// AttachSignatureToImageIndex attaches the provided signature to the provided image index.
func AttachSignatureToImageIndex(sii oci.SignedImageIndex, sig oci.Signature, opts ...SignOption) (oci.SignedImageIndex, error) {
	return &signedImageIndex{
		ociSignedImageIndex: sii,
		sig:                 sig,
		attachments:         make(map[string]oci.File),
		so:                  makeSignOpts(opts...),
	}, nil
}

// AttachAttestationToImageIndex attaches the provided attestation to the provided image index.
func AttachAttestationToImageIndex(sii oci.SignedImageIndex, att oci.Signature, opts ...SignOption) (oci.SignedImageIndex, error) {
	return &signedImageIndex{
		ociSignedImageIndex: sii,
		att:                 att,
		attachments:         make(map[string]oci.File),
		so:                  makeSignOpts(opts...),
	}, nil
}

// AttachFileToImageIndex attaches the provided file to the provided image index.
func AttachFileToImageIndex(sii oci.SignedImageIndex, name string, f oci.File, opts ...SignOption) (oci.SignedImageIndex, error) {
	return &signedImageIndex{
		ociSignedImageIndex: sii,
		attachments: map[string]oci.File{
			name: f,
		},
		so: makeSignOpts(opts...),
	}, nil
}

type ociSignedImageIndex oci.SignedImageIndex

type signedImageIndex struct {
	ociSignedImageIndex
	sig         oci.Signature
	att         oci.Signature
	so          *signOpts
	attachments map[string]oci.File
}

// Signatures implements oci.SignedImageIndex
func (sii *signedImageIndex) Signatures() (oci.Signatures, error) {
	return sii.so.dedupeAndReplace(sii.sig, sii.ociSignedImageIndex.Signatures)
}

// Attestations implements oci.SignedImageIndex
func (sii *signedImageIndex) Attestations() (oci.Signatures, error) {
	return sii.so.dedupeAndReplace(sii.att, sii.ociSignedImageIndex.Attestations)
}

// Attachment implements oci.SignedImageIndex
func (sii *signedImageIndex) Attachment(attName string) (oci.File, error) {
	if f, ok := sii.attachments[attName]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("attachment %q not found", attName)
}

// AttachSignatureToUnknown attaches the provided signature to the provided image.
func AttachSignatureToUnknown(se oci.SignedEntity, sig oci.Signature, opts ...SignOption) (oci.SignedEntity, error) {
	return &signedUnknown{
		SignedEntity: se,
		sig:          sig,
		attachments:  make(map[string]oci.File),
		so:           makeSignOpts(opts...),
	}, nil
}

// AttachAttestationToUnknown attaches the provided attestation to the provided image.
func AttachAttestationToUnknown(se oci.SignedEntity, att oci.Signature, opts ...SignOption) (oci.SignedEntity, error) {
	return &signedUnknown{
		SignedEntity: se,
		att:          att,
		attachments:  make(map[string]oci.File),
		so:           makeSignOpts(opts...),
	}, nil
}

// AttachFileToUnknown attaches the provided file to the provided image.
func AttachFileToUnknown(se oci.SignedEntity, name string, f oci.File, opts ...SignOption) (oci.SignedEntity, error) {
	return &signedUnknown{
		SignedEntity: se,
		attachments: map[string]oci.File{
			name: f,
		},
		so: makeSignOpts(opts...),
	}, nil
}

type signedUnknown struct {
	oci.SignedEntity
	sig         oci.Signature
	att         oci.Signature
	so          *signOpts
	attachments map[string]oci.File
}

type digestable interface {
	Digest() (v1.Hash, error)
}

// Digest is generally implemented by oci.SignedEntity implementations.
func (si *signedUnknown) Digest() (v1.Hash, error) {
	d, ok := si.SignedEntity.(digestable)
	if !ok {
		return v1.Hash{}, fmt.Errorf("underlying signed entity not digestable: %T", si.SignedEntity)
	}
	return d.Digest()
}

// Signatures implements oci.SignedEntity
func (si *signedUnknown) Signatures() (oci.Signatures, error) {
	return si.so.dedupeAndReplace(si.sig, si.SignedEntity.Signatures)
}

// Attestations implements oci.SignedEntity
func (si *signedUnknown) Attestations() (oci.Signatures, error) {
	return si.so.dedupeAndReplace(si.att, si.SignedEntity.Attestations)
}

// Attachment implements oci.SignedEntity
func (si *signedUnknown) Attachment(attName string) (oci.File, error) {
	if f, ok := si.attachments[attName]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("attachment %q not found", attName)
}

func (so *signOpts) dedupeAndReplace(sig oci.Signature, basefn func() (oci.Signatures, error)) (oci.Signatures, error) {
	base, err := basefn()
	if err != nil {
		return nil, err
	} else if sig == nil {
		return base, nil
	}
	if so.dd != nil {
		if existing, err := so.dd.Find(base, sig); err != nil {
			return nil, err
		} else if existing != nil {
			// Just return base if the signature is redundant
			return base, nil
		}
	}
	if so.ro != nil {
		replace, err := so.ro.Replace(base, sig)
		if err != nil {
			return nil, err
		}
		return ReplaceSignatures(replace)
	}
	return AppendSignatures(base, sig)
}
