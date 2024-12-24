//
// Copyright 2023 The Sigstore Authors.
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
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

// SignedUnknown provides access to signed metadata without directly accessing
// the underlying entity.  This can be used to access signature metadata for
// digests that have not been published (yet).
func SignedUnknown(digest name.Digest, options ...Option) oci.SignedEntity {
	o := makeOptions(digest.Context(), options...)
	return &unknown{
		digest: digest,
		opt:    o,
	}
}

type unknown struct {
	digest name.Digest
	opt    *options
}

var _ oci.SignedEntity = (*unknown)(nil)

// Digest implements digestable
func (i *unknown) Digest() (v1.Hash, error) {
	return v1.NewHash(i.digest.DigestStr())
}

// Signatures implements oci.SignedEntity
func (i *unknown) Signatures() (oci.Signatures, error) {
	return signatures(i, i.opt)
}

// Attestations implements oci.SignedEntity
func (i *unknown) Attestations() (oci.Signatures, error) {
	return attestations(i, i.opt)
}

// Attachment implements oci.SignedEntity
func (i *unknown) Attachment(name string) (oci.File, error) {
	return attachment(i, name, i.opt)
}
