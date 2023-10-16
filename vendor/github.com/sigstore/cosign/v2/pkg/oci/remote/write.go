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

package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	ociexperimental "github.com/sigstore/cosign/v2/internal/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ctypes "github.com/sigstore/cosign/v2/pkg/types"
)

// WriteSignedImageIndexImages writes the images within the image index
// This includes the signed image and associated signatures in the image index
// TODO (priyawadhwa@): write the `index.json` itself to the repo as well
// TODO (priyawadhwa@): write the attestations
func WriteSignedImageIndexImages(ref name.Reference, sii oci.SignedImageIndex, opts ...Option) error {
	repo := ref.Context()
	o := makeOptions(repo, opts...)

	// write the image index if there is one
	ii, err := sii.SignedImageIndex(v1.Hash{})
	if err != nil {
		return fmt.Errorf("signed image index: %w", err)
	}
	if ii != nil {
		if err := remote.WriteIndex(ref, ii, o.ROpt...); err != nil {
			return fmt.Errorf("writing index: %w", err)
		}
	}

	// write the image if there is one
	si, err := sii.SignedImage(v1.Hash{})
	if err != nil {
		return fmt.Errorf("signed image: %w", err)
	}
	if si != nil {
		if err := remoteWrite(ref, si, o.ROpt...); err != nil {
			return fmt.Errorf("remote write: %w", err)
		}
	}

	// write the signatures
	sigs, err := sii.Signatures()
	if err != nil {
		return err
	}
	if sigs != nil { // will be nil if there are no associated signatures
		sigsTag, err := SignatureTag(ref, opts...)
		if err != nil {
			return fmt.Errorf("sigs tag: %w", err)
		}
		if err := remoteWrite(sigsTag, sigs, o.ROpt...); err != nil {
			return err
		}
	}

	// write the attestations
	atts, err := sii.Attestations()
	if err != nil {
		return err
	}
	if atts != nil { // will be nil if there are no associated attestations
		attsTag, err := AttestationTag(ref, opts...)
		if err != nil {
			return fmt.Errorf("sigs tag: %w", err)
		}
		return remoteWrite(attsTag, atts, o.ROpt...)
	}
	return nil
}

// WriteSignature publishes the signatures attached to the given entity
// into the provided repository.
func WriteSignatures(repo name.Repository, se oci.SignedEntity, opts ...Option) error {
	o := makeOptions(repo, opts...)

	// Access the signature list to publish
	sigs, err := se.Signatures()
	if err != nil {
		return err
	}

	// Determine the tag to which these signatures should be published.
	h, err := se.Digest()
	if err != nil {
		return err
	}
	tag := o.TargetRepository.Tag(normalize(h, o.TagPrefix, o.SignatureSuffix))

	// Write the Signatures image to the tag, with the provided remote.Options
	return remoteWrite(tag, sigs, o.ROpt...)
}

// WriteAttestations publishes the attestations attached to the given entity
// into the provided repository.
func WriteAttestations(repo name.Repository, se oci.SignedEntity, opts ...Option) error {
	o := makeOptions(repo, opts...)

	// Access the signature list to publish
	atts, err := se.Attestations()
	if err != nil {
		return err
	}

	// Determine the tag to which these signatures should be published.
	h, err := se.Digest()
	if err != nil {
		return err
	}
	tag := o.TargetRepository.Tag(normalize(h, o.TagPrefix, o.AttestationSuffix))

	// Write the Signatures image to the tag, with the provided remote.Options
	return remoteWrite(tag, atts, o.ROpt...)
}

// WriteSignaturesExperimentalOCI publishes the signatures attached to the given entity
// into the provided repository (using OCI 1.1 methods).
func WriteSignaturesExperimentalOCI(d name.Digest, se oci.SignedEntity, opts ...Option) error {
	o := makeOptions(d.Repository, opts...)
	signTarget := d.String()
	ref, err := name.ParseReference(signTarget, o.NameOpts...)
	if err != nil {
		return err
	}
	desc, err := remote.Head(ref, o.ROpt...)
	if err != nil {
		return err
	}
	sigs, err := se.Signatures()
	if err != nil {
		return err
	}

	// Write the signature blobs
	s, err := sigs.Get()
	if err != nil {
		return err
	}
	for _, v := range s {
		if err := remote.WriteLayer(d.Repository, v, o.ROpt...); err != nil {
			return err
		}
	}

	// Write the config
	configBytes, err := sigs.RawConfigFile()
	if err != nil {
		return err
	}
	var configDesc v1.Descriptor
	if err := json.Unmarshal(configBytes, &configDesc); err != nil {
		return err
	}
	configLayer := static.NewLayer(configBytes, configDesc.MediaType)
	if err := remote.WriteLayer(d.Repository, configLayer, o.ROpt...); err != nil {
		return err
	}

	// Write the manifest containing a subject
	b, err := sigs.RawManifest()
	if err != nil {
		return err
	}
	var m v1.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	artifactType := ociexperimental.ArtifactType("sig")
	m.Config.MediaType = types.MediaType(artifactType)
	m.Subject = desc
	b, err = json.Marshal(&m)
	if err != nil {
		return err
	}
	digest, _, err := v1.SHA256(bytes.NewReader(b))
	if err != nil {
		return err
	}
	targetRef, err := name.ParseReference(fmt.Sprintf("%s/%s@%s", d.RegistryStr(), d.RepositoryStr(), digest.String()))
	if err != nil {
		return err
	}
	// TODO: use ui.Infof
	fmt.Fprintf(os.Stderr, "Uploading signature for [%s] to [%s] with config.mediaType [%s] layers[0].mediaType [%s].\n",
		d.String(), targetRef.String(), artifactType, ctypes.SimpleSigningMediaType)
	return remote.Put(targetRef, &taggableManifest{raw: b, mediaType: m.MediaType}, o.ROpt...)
}

type taggableManifest struct {
	raw       []byte
	mediaType types.MediaType
}

func (taggable taggableManifest) RawManifest() ([]byte, error) {
	return taggable.raw, nil
}

func (taggable taggableManifest) MediaType() (types.MediaType, error) {
	return taggable.mediaType, nil
}
