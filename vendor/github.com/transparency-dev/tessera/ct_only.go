// Copyright 2024 The Tessera authors. All Rights Reserved.
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

package tessera

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/ctonly"
	"golang.org/x/crypto/cryptobyte"
)

// NewCertificateTransparencyAppender returns a function which knows how to add a CT-specific entry type to the log.
//
// This entry point MUST ONLY be used for CT logs participating in the CT ecosystem.
// It should not be used as the basis for any other/new transparency application as this protocol:
// a) embodies some techniques which are not considered to be best practice (it does this to retain backawards-compatibility with RFC6962)
// b) is not compatible with the https://c2sp.org/tlog-tiles API which we _very strongly_ encourage you to use instead.
//
// Users of this MUST NOT call `Add` on the underlying Appender directly.
//
// Returns a future, which resolves to the assigned index in the log, or an error.
func NewCertificateTransparencyAppender(a *Appender) func(context.Context, *ctonly.Entry) IndexFuture {
	return func(ctx context.Context, e *ctonly.Entry) IndexFuture {
		return a.Add(ctx, convertCTEntry(e))
	}
}

// convertCTEntry returns an Entry struct which will do the right thing for CT Static API logs.
//
// This MUST NOT be used for any other purpose.
func convertCTEntry(e *ctonly.Entry) *Entry {
	r := &Entry{}
	r.internal.Identity = e.Identity()
	r.marshalForBundle = func(idx uint64) []byte {
		r.internal.LeafHash = e.MerkleLeafHash(idx)
		r.internal.Data = e.LeafData(idx)
		return r.internal.Data
	}

	return r
}

// WithCTLayout instructs the underlying storage to use a Static CT API compatible scheme for layout.
func (o *AppendOptions) WithCTLayout() *AppendOptions {
	o.entriesPath = ctEntriesPath
	o.bundleIDHasher = ctBundleIDHasher
	return o
}

// WithCTLayout instructs the underlying storage to use a Static CT API compatible scheme for layout.
func (o *MigrationOptions) WithCTLayout() *MigrationOptions {
	o.entriesPath = ctEntriesPath
	o.bundleIDHasher = ctBundleIDHasher
	o.bundleLeafHasher = ctMerkleLeafHasher
	return o
}

func ctEntriesPath(n uint64, p uint8) string {
	return fmt.Sprintf("tile/data/%s", layout.NWithSuffix(0, n, p))
}

// ctBundleIDHasher knows how to calculate antispam identity hashes for entries in a Static-CT formatted entry bundle.
func ctBundleIDHasher(bundle []byte) ([][]byte, error) {
	r := make([][]byte, 0, layout.EntryBundleWidth)
	b := cryptobyte.String(bundle)
	for i := 0; i < layout.EntryBundleWidth && !b.Empty(); i++ {
		// Timestamp
		if !b.Skip(8) {
			return nil, fmt.Errorf("failed to read timestamp of entry index %d of bundle", i)
		}

		var entryType uint16
		if !b.ReadUint16(&entryType) {
			return nil, fmt.Errorf("failed to read entry type of entry index %d of bundle", i)
		}

		switch entryType {
		case 0: // X509 entry
			cert := cryptobyte.String{}
			if !b.ReadUint24LengthPrefixed(&cert) {
				return nil, fmt.Errorf("failed to read certificate at entry index %d of bundle", i)
			}

			// For x509 entries we hash (just) the x509 certificate for identity.
			r = append(r, identityHash(cert))

			// Must continue below to consume all the remaining bytes in the entry.

		case 1: // Precert entry
			// IssuerKeyHash
			if !b.Skip(sha256.Size) {
				return nil, fmt.Errorf("failed to read issuer key hash at entry index %d of bundle", i)
			}
			tbs := cryptobyte.String{}
			if !b.ReadUint24LengthPrefixed(&tbs) {
				return nil, fmt.Errorf("failed to read precert tbs at entry index %d of bundle", i)
			}

		default:
			return nil, fmt.Errorf("unknown entry type at entry index %d of bundle", i)
		}

		ignore := cryptobyte.String{}
		if !b.ReadUint16LengthPrefixed(&ignore) {
			return nil, fmt.Errorf("failed to read SCT extensions at entry index %d of bundle", i)
		}

		if entryType == 1 {
			precert := cryptobyte.String{}
			if !b.ReadUint24LengthPrefixed(&precert) {
				return nil, fmt.Errorf("failed to read precert at entry index %d of bundle", i)
			}
			// For Precert entries we hash (just) the full precertificate for identity.
			r = append(r, identityHash(precert))

		}
		if !b.ReadUint16LengthPrefixed(&ignore) {
			return nil, fmt.Errorf("failed to read chain fingerprints at entry index %d of bundle", i)
		}
	}
	if !b.Empty() {
		return nil, fmt.Errorf("unexpected %d bytes of trailing data in entry bundle", len(b))
	}
	return r, nil
}

// copyBytes copies N bytes between from and to.
func copyBytes(from *cryptobyte.String, to *cryptobyte.Builder, N int) bool {
	b := make([]byte, N)
	if !from.ReadBytes(&b, N) {
		return false
	}
	to.AddBytes(b)
	return true
}

// copyUint16LengthPrefixed copies a uint16 length and value between from and to.
func copyUint16LengthPrefixed(from *cryptobyte.String, to *cryptobyte.Builder) bool {
	b := cryptobyte.String{}
	if !from.ReadUint16LengthPrefixed(&b) {
		return false
	}
	to.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) {
		c.AddBytes(b)
	})
	return true
}

// copyUint24LengthPrefixed copies a uint24 length and value between from and to.
func copyUint24LengthPrefixed(from *cryptobyte.String, to *cryptobyte.Builder) bool {
	b := cryptobyte.String{}
	if !from.ReadUint24LengthPrefixed(&b) {
		return false
	}
	to.AddUint24LengthPrefixed(func(c *cryptobyte.Builder) {
		c.AddBytes(b)
	})
	return true
}

// ctMerkleLeafHasher knows how to calculate RFC6962 Merkle leaf hashes for entries in a Static-CT formatted entry bundle.
func ctMerkleLeafHasher(bundle []byte) ([][]byte, error) {
	r := make([][]byte, 0, layout.EntryBundleWidth)
	b := cryptobyte.String(bundle)
	for i := 0; i < layout.EntryBundleWidth && !b.Empty(); i++ {
		preimage := &cryptobyte.Builder{}
		preimage.AddUint8(0 /* version = v1 */)
		preimage.AddUint8(0 /* leaf_type = timestamped_entry */)

		// Timestamp
		if !copyBytes(&b, preimage, 8) {
			return nil, fmt.Errorf("failed to copy timestamp of entry index %d of bundle", i)
		}

		var entryType uint16
		if !b.ReadUint16(&entryType) {
			return nil, fmt.Errorf("failed to read entry type of entry index %d of bundle", i)
		}
		preimage.AddUint16(entryType)

		switch entryType {
		case 0: // X509 entry
			if !copyUint24LengthPrefixed(&b, preimage) {
				return nil, fmt.Errorf("failed to copy certificate at entry index %d of bundle", i)
			}

		case 1: // Precert entry
			// IssuerKeyHash
			if !copyBytes(&b, preimage, sha256.Size) {
				return nil, fmt.Errorf("failed to copy issuer key hash at entry index %d of bundle", i)
			}

			if !copyUint24LengthPrefixed(&b, preimage) {
				return nil, fmt.Errorf("failed to copy precert tbs at entry index %d of bundle", i)
			}

		default:
			return nil, fmt.Errorf("unknown entry type 0x%x at entry index %d of bundle", entryType, i)
		}

		if !copyUint16LengthPrefixed(&b, preimage) {
			return nil, fmt.Errorf("failed to copy SCT extensions at entry index %d of bundle", i)
		}

		ignore := cryptobyte.String{}
		if entryType == 1 {
			if !b.ReadUint24LengthPrefixed(&ignore) {
				return nil, fmt.Errorf("failed to read precert at entry index %d of bundle", i)
			}
		}
		if !b.ReadUint16LengthPrefixed(&ignore) {
			return nil, fmt.Errorf("failed to read chain fingerprints at entry index %d of bundle", i)
		}

		h := rfc6962.DefaultHasher.HashLeaf(preimage.BytesOrPanic())
		r = append(r, h)
	}
	if !b.Empty() {
		return nil, fmt.Errorf("unexpected %d bytes of trailing data in entry bundle", len(b))
	}
	return r, nil
}
