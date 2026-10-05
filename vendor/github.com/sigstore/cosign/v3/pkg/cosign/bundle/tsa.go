// Copyright 2022 The Sigstore Authors.
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

// RFC3161Timestamp holds metadata about timestamp RFC3161 verification data.
type RFC3161Timestamp struct {
	// SignedRFC3161Timestamp contains a DER encoded TimeStampResponse.
	// See https://www.rfc-editor.org/rfc/rfc3161.html#section-2.4.2
	// Clients MUST verify the hashed message in the message imprint,
	// typically using the artifact signature.
	SignedRFC3161Timestamp []byte
}

// TimestampToRFC3161Timestamp receives a base64 encoded RFC3161 timestamp.
func TimestampToRFC3161Timestamp(timestampRFC3161 []byte) *RFC3161Timestamp {
	if timestampRFC3161 != nil {
		return &RFC3161Timestamp{
			SignedRFC3161Timestamp: timestampRFC3161,
		}
	}
	return nil
}
