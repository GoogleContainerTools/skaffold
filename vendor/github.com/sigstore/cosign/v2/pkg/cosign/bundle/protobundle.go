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

package bundle

import (
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/tle"
)

const bundleV03MediaType = "application/vnd.dev.sigstore.bundle.v0.3+json"

func MakeProtobufBundle(hint string, rawCert []byte, rekorEntry *models.LogEntryAnon, timestampBytes []byte) (*protobundle.Bundle, error) {
	bundle := &protobundle.Bundle{MediaType: bundleV03MediaType}

	if hint != "" {
		bundle.VerificationMaterial = &protobundle.VerificationMaterial{
			Content: &protobundle.VerificationMaterial_PublicKey{
				PublicKey: &protocommon.PublicKeyIdentifier{
					Hint: hint,
				},
			},
		}
	} else if len(rawCert) > 0 {
		bundle.VerificationMaterial = &protobundle.VerificationMaterial{
			Content: &protobundle.VerificationMaterial_Certificate{
				Certificate: &protocommon.X509Certificate{
					RawBytes: rawCert,
				},
			},
		}
	}

	if len(timestampBytes) > 0 {
		bundle.VerificationMaterial.TimestampVerificationData = &protobundle.TimestampVerificationData{
			Rfc3161Timestamps: []*protocommon.RFC3161SignedTimestamp{
				{SignedTimestamp: timestampBytes},
			},
		}
	}

	if rekorEntry != nil {
		tlogEntry, err := tle.GenerateTransparencyLogEntry(*rekorEntry)
		if err != nil {
			return nil, err
		}
		bundle.VerificationMaterial.TlogEntries = []*protorekor.TransparencyLogEntry{tlogEntry}
	}

	return bundle, nil
}
