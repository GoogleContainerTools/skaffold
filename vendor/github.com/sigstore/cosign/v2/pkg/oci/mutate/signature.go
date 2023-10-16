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
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

type sigWrapper struct {
	wrapped oci.Signature

	annotations      map[string]string
	bundle           *bundle.RekorBundle
	rfc3161Timestamp *bundle.RFC3161Timestamp
	cert             *x509.Certificate
	chain            []*x509.Certificate
	mediaType        types.MediaType
}

var _ v1.Layer = (*sigWrapper)(nil)
var _ oci.Signature = (*sigWrapper)(nil)

func copyAnnotations(ann map[string]string) map[string]string {
	new := make(map[string]string, len(ann)) //nolint: revive
	for k, v := range ann {
		new[k] = v
	}
	return new
}

// Annotations implements oci.Signature.
func (sw *sigWrapper) Annotations() (map[string]string, error) {
	if sw.annotations != nil {
		return copyAnnotations(sw.annotations), nil
	}
	return sw.wrapped.Annotations()
}

// Payload implements oci.Signature.
func (sw *sigWrapper) Payload() ([]byte, error) {
	return sw.wrapped.Payload()
}

// Signature implements oci.Signature
func (sw *sigWrapper) Signature() ([]byte, error) {
	return sw.wrapped.Signature()
}

// Base64Signature implements oci.Signature.
func (sw *sigWrapper) Base64Signature() (string, error) {
	return sw.wrapped.Base64Signature()
}

// Cert implements oci.Signature.
func (sw *sigWrapper) Cert() (*x509.Certificate, error) {
	if sw.cert != nil {
		return sw.cert, nil
	}
	return sw.wrapped.Cert()
}

// Chain implements oci.Signature.
func (sw *sigWrapper) Chain() ([]*x509.Certificate, error) {
	if sw.chain != nil {
		return sw.chain, nil
	}
	return sw.wrapped.Chain()
}

// Bundle implements oci.Signature.
func (sw *sigWrapper) Bundle() (*bundle.RekorBundle, error) {
	if sw.bundle != nil {
		return sw.bundle, nil
	}
	return sw.wrapped.Bundle()
}

// RFC3161Timestamp implements oci.Signature.
func (sw *sigWrapper) RFC3161Timestamp() (*bundle.RFC3161Timestamp, error) {
	if sw.rfc3161Timestamp != nil {
		return sw.rfc3161Timestamp, nil
	}
	return sw.wrapped.RFC3161Timestamp()
}

// MediaType implements v1.Layer
func (sw *sigWrapper) MediaType() (types.MediaType, error) {
	if sw.mediaType != "" {
		return sw.mediaType, nil
	}
	return sw.wrapped.MediaType()
}

// Digest implements v1.Layer
func (sw *sigWrapper) Digest() (v1.Hash, error) {
	return sw.wrapped.Digest()
}

// DiffID implements v1.Layer
func (sw *sigWrapper) DiffID() (v1.Hash, error) {
	return sw.wrapped.DiffID()
}

// Compressed implements v1.Layer
func (sw *sigWrapper) Compressed() (io.ReadCloser, error) {
	return sw.wrapped.Compressed()
}

// Uncompressed implements v1.Layer
func (sw *sigWrapper) Uncompressed() (io.ReadCloser, error) {
	return sw.wrapped.Uncompressed()
}

// Size implements v1.Layer
func (sw *sigWrapper) Size() (int64, error) {
	return sw.wrapped.Size()
}

// Signature returns a new oci.Signature based on the provided original, plus the requested mutations.
func Signature(original oci.Signature, opts ...SignatureOption) (oci.Signature, error) {
	newSig := sigWrapper{wrapped: original}

	so := makeSignatureOption(opts...)
	oldAnn, err := original.Annotations()
	if err != nil {
		return nil, fmt.Errorf("could not get annotations from signature to mutate: %w", err)
	}

	var newAnn map[string]string
	if so.annotations != nil {
		newAnn = copyAnnotations(so.annotations)
		newAnn[static.SignatureAnnotationKey] = oldAnn[static.SignatureAnnotationKey]
		for _, key := range []string{static.BundleAnnotationKey, static.CertificateAnnotationKey, static.ChainAnnotationKey, static.RFC3161TimestampAnnotationKey} {
			if val, isSet := oldAnn[key]; isSet {
				newAnn[key] = val
			} else {
				delete(newAnn, key)
			}
		}
	} else {
		newAnn = copyAnnotations(oldAnn)
	}

	if so.bundle != nil {
		newSig.bundle = so.bundle
		b, err := json.Marshal(so.bundle)
		if err != nil {
			return nil, err
		}
		newAnn[static.BundleAnnotationKey] = string(b)
	}

	if so.rfc3161Timestamp != nil {
		newSig.rfc3161Timestamp = so.rfc3161Timestamp
		b, err := json.Marshal(so.rfc3161Timestamp)
		if err != nil {
			return nil, err
		}
		newAnn[static.RFC3161TimestampAnnotationKey] = string(b)
	}

	if so.cert != nil {
		var cert *x509.Certificate
		var chain []*x509.Certificate

		certs, err := cryptoutils.LoadCertificatesFromPEM(bytes.NewReader(so.cert))
		if err != nil {
			return nil, err
		}
		newAnn[static.CertificateAnnotationKey] = string(so.cert)
		cert = certs[0]

		delete(newAnn, static.ChainAnnotationKey)
		if so.chain != nil {
			chain, err = cryptoutils.LoadCertificatesFromPEM(bytes.NewReader(so.chain))
			if err != nil {
				return nil, err
			}
			newAnn[static.ChainAnnotationKey] = string(so.chain)
		}

		newSig.cert = cert
		newSig.chain = chain
	}

	if so.mediaType != "" {
		newSig.mediaType = so.mediaType
	}

	newSig.annotations = newAnn

	return &newSig, nil
}
