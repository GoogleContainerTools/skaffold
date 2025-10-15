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

// Package tessera provides an implementation of a tile-based logging framework.
package tessera

import (
	"encoding/binary"

	"github.com/transparency-dev/merkle/rfc6962"
)

// Entry represents an entry in a log.
type Entry struct {
	// We keep the all data in exported fields inside an unexported interal struct.
	// This allows us to use gob to serialise the entry data (relying on the backwards-compatibility
	// it provides), while also keeping these fields private which allows us to deter bad practice
	// by forcing use of the API to set these values to safe values.
	internal struct {
		Data     []byte
		Identity []byte
		LeafHash []byte
		Index    *uint64
	}

	// marshalForBundle knows how to convert this entry's Data into a marshalled bundle entry.
	marshalForBundle func(index uint64) []byte
}

// Data returns the raw entry bytes which will form the entry in the log.
func (e Entry) Data() []byte { return e.internal.Data }

// Identity returns an identity which may be used to de-duplicate entries and they are being added to the log.
func (e Entry) Identity() []byte { return e.internal.Identity }

// LeafHash is the Merkle leaf hash which will be used for this entry in the log.
// Note that in almost all cases, this should be the RFC6962 definition of a leaf hash.
func (e Entry) LeafHash() []byte { return e.internal.LeafHash }

// Index returns the index assigned to the entry in the log, or nil if no index has been assigned.
func (e Entry) Index() *uint64 { return e.internal.Index }

// MarshalBundleData returns this entry's data in a format ready to be appended to an EntryBundle.
//
// Note that MarshalBundleData _may_ be called multiple times, potentially with different values for index
// (e.g. if there's a failure in the storage when trying to persist the assignment), so index should not
// be considered final until the storage Add method has returned successfully with the durably assigned index.
func (e *Entry) MarshalBundleData(index uint64) []byte {
	e.internal.Index = &index
	return e.marshalForBundle(index)
}

// NewEntry creates a new Entry object with leaf data.
func NewEntry(data []byte) *Entry {
	e := &Entry{}
	e.internal.Data = data
	h := identityHash(e.internal.Data)
	e.internal.Identity = h[:]
	e.internal.LeafHash = rfc6962.DefaultHasher.HashLeaf(e.internal.Data)
	// By default we will marshal ourselves into a bundle using the mechanism described
	// by https://c2sp.org/tlog-tiles:
	e.marshalForBundle = func(_ uint64) []byte {
		r := make([]byte, 0, 2+len(e.internal.Data))
		r = binary.BigEndian.AppendUint16(r, uint16(len(e.internal.Data)))
		r = append(r, e.internal.Data...)
		return r
	}
	return e
}
