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
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

// DupeDetector scans a list of signatures looking for a duplicate.
type DupeDetector interface {
	Find(oci.Signatures, oci.Signature) (oci.Signature, error)
}

type ReplaceOp interface {
	Replace(oci.Signatures, oci.Signature) (oci.Signatures, error)
}

type SignOption func(*signOpts)

type signOpts struct {
	dd DupeDetector
	ro ReplaceOp
}

func makeSignOpts(opts ...SignOption) *signOpts {
	so := &signOpts{}
	for _, opt := range opts {
		opt(so)
	}
	return so
}

// WithDupeDetector configures Sign* to use the following DupeDetector
// to avoid attaching duplicate signatures.
func WithDupeDetector(dd DupeDetector) SignOption {
	return func(so *signOpts) {
		so.dd = dd
	}
}

func WithReplaceOp(ro ReplaceOp) SignOption {
	return func(so *signOpts) {
		so.ro = ro
	}
}

type signatureOpts struct {
	annotations      map[string]string
	bundle           *bundle.RekorBundle
	rfc3161Timestamp *bundle.RFC3161Timestamp
	cert             []byte
	chain            []byte
	mediaType        types.MediaType
}

type SignatureOption func(*signatureOpts)

// WithAnnotations specifies the annotations the Signature should have.
func WithAnnotations(annotations map[string]string) SignatureOption {
	return func(so *signatureOpts) {
		so.annotations = annotations
	}
}

// WithBundle specifies the new Bundle the Signature should have.
func WithBundle(b *bundle.RekorBundle) SignatureOption {
	return func(so *signatureOpts) {
		so.bundle = b
	}
}

// WithRFC3161Timestamp specifies the new RFC3161Timestamp the Signature should have.
func WithRFC3161Timestamp(b *bundle.RFC3161Timestamp) SignatureOption {
	return func(so *signatureOpts) {
		so.rfc3161Timestamp = b
	}
}

// WithCertChain specifies the new cert and chain the Signature should have.
func WithCertChain(cert, chain []byte) SignatureOption {
	return func(so *signatureOpts) {
		so.cert = cert
		so.chain = chain
	}
}

// WithMediaType specifies the new MediaType the Signature should have.
func WithMediaType(mediaType types.MediaType) SignatureOption {
	return func(so *signatureOpts) {
		so.mediaType = mediaType
	}
}

func makeSignatureOption(opts ...SignatureOption) *signatureOpts {
	so := &signatureOpts{}
	for _, opt := range opts {
		opt(so)
	}
	return so
}
