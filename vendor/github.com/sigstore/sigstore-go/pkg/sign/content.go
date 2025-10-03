// Copyright 2024 The Sigstore Authors.
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

package sign

import (
	"fmt"

	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	protodsse "github.com/sigstore/protobuf-specs/gen/pb-go/dsse"
)

type Content interface {
	// Return the data to be signed
	PreAuthEncoding() []byte
	// Add something that satisfies protobundle.isBundle_Content to bundle
	Bundle(bundle *protobundle.Bundle, signature, digest []byte, hashAlgorithm protocommon.HashAlgorithm)
}

type PlainData struct {
	Data []byte
}

func (pd *PlainData) PreAuthEncoding() []byte {
	return pd.Data
}

func (pd *PlainData) Bundle(bundle *protobundle.Bundle, signature, digest []byte, hashAlgorithm protocommon.HashAlgorithm) {
	bundle.Content = &protobundle.Bundle_MessageSignature{
		MessageSignature: &protocommon.MessageSignature{
			MessageDigest: &protocommon.HashOutput{
				Algorithm: hashAlgorithm,
				Digest:    digest,
			},
			Signature: signature,
		},
	}
}

type DSSEData struct {
	Data        []byte
	PayloadType string
}

func (d *DSSEData) PreAuthEncoding() []byte {
	pae := fmt.Sprintf("DSSEv1 %d %s %d %s", len(d.PayloadType), d.PayloadType, len(d.Data), d.Data)
	return []byte(pae)
}

func (d *DSSEData) Bundle(bundle *protobundle.Bundle, signature, _ []byte, _ protocommon.HashAlgorithm) {
	sig := &protodsse.Signature{
		Sig: signature,
	}

	bundle.Content = &protobundle.Bundle_DsseEnvelope{
		DsseEnvelope: &protodsse.Envelope{
			Payload:     []byte(d.Data),
			PayloadType: d.PayloadType,
			Signatures:  []*protodsse.Signature{sig},
		},
	}
}
