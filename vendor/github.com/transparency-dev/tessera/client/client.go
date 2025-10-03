// Copyright 2024 Google LLC. All Rights Reserved.
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

// Package client provides client support for interacting with logs that
// uses the [tlog-tiles API].
//
// [tlog-tiles API]: https://c2sp.org/tlog-tiles
package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/transparency-dev/formats/log"
	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/internal/otel"
	"golang.org/x/mod/sumdb/note"
)

var (
	hasher = rfc6962.DefaultHasher
)

// ErrInconsistency should be returned when there has been an error proving consistency
// between log states.
// The raw log state representations are included as-returned by the target log, this
// ensures that evidence of inconsistent log updates are available to the caller of
// the method(s) returning this error.
type ErrInconsistency struct {
	SmallerRaw []byte
	LargerRaw  []byte
	Proof      [][]byte

	Wrapped error
}

func (e ErrInconsistency) Unwrap() error {
	return e.Wrapped
}

func (e ErrInconsistency) Error() string {
	return fmt.Sprintf("log consistency check failed: %s", e.Wrapped)
}

// CheckpointFetcherFunc is the signature of a function which can retrieve the latest
// checkpoint from a log's data storage.
//
// Note that the implementation of this MUST return (either directly or wrapped)
// an os.ErrIsNotExist when the file referenced by path does not exist, e.g. a HTTP
// based implementation MUST return this error when it receives a 404 StatusCode.
type CheckpointFetcherFunc func(ctx context.Context) ([]byte, error)

// TileFetcherFunc is the signature of a function which can fetch the raw data
// for a given tile.
//
// Note that the implementation of this MUST:
//   - when asked to fetch a partial tile (i.e. p != 0), fall-back to fetching the corresponding full
//     tile if the partial one does not exist.
//   - return (either directly or wrapped) an os.ErrIsNotExist when neither the requested tile nor any
//     fallback tile exists.
type TileFetcherFunc func(ctx context.Context, level, index uint64, p uint8) ([]byte, error)

// EntryBundleFetcherFunc is the signature of a function which can fetch the raw data
// for a given entry bundle.
//
// Note that the implementation of this MUST:
//   - when asked to fetch a partial entry bundle (i.e. p != 0), fall-back to fetching the corresponding full
//     bundle if the partial one does not exist.
//   - return (either directly or wrapped) an os.ErrIsNotExist when neither the requested bundle nor any
//     fallback bundle exists.
type EntryBundleFetcherFunc func(ctx context.Context, bundleIndex uint64, p uint8) ([]byte, error)

// ConsensusCheckpointFunc is a function which returns the largest checkpoint known which is
// signed by logSigV and satisfies some consensus algorithm.
//
// This is intended to provide a hook for adding a consensus view of a log, e.g. via witnessing.
type ConsensusCheckpointFunc func(ctx context.Context, logSigV note.Verifier, origin string) (*log.Checkpoint, []byte, *note.Note, error)

// UnilateralConsensus blindly trusts the source log, returning the checkpoint it provided.
func UnilateralConsensus(f CheckpointFetcherFunc) ConsensusCheckpointFunc {
	return func(ctx context.Context, logSigV note.Verifier, origin string) (*log.Checkpoint, []byte, *note.Note, error) {
		return FetchCheckpoint(ctx, f, logSigV, origin)
	}
}

// FetchCheckpoint retrieves and opens a checkpoint from the log.
// Returns both the parsed structure and the raw serialised checkpoint.
func FetchCheckpoint(ctx context.Context, f CheckpointFetcherFunc, v note.Verifier, origin string) (*log.Checkpoint, []byte, *note.Note, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.FetchCheckpoint")
	defer span.End()

	cpRaw, err := f(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	cp, _, n, err := log.ParseCheckpoint(cpRaw, origin, v)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse Checkpoint: %v", err)
	}
	return cp, cpRaw, n, nil
}

// FetchRangeNodes returns the set of nodes representing the compact range covering
// a log of size s.
func FetchRangeNodes(ctx context.Context, s uint64, f TileFetcherFunc) ([][]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.FetchRangeNodes")
	defer span.End()
	span.SetAttributes(logSizeKey.Int64(otel.Clamp64(s)))

	nc := newNodeCache(f, s)
	nIDs := make([]compact.NodeID, 0, compact.RangeSize(0, s))
	nIDs = compact.RangeNodes(0, s, nIDs)
	hashes := make([][]byte, 0, len(nIDs))
	for _, n := range nIDs {
		h, err := nc.GetNode(ctx, n)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}
	return hashes, nil
}

// FetchLeafHashes fetches N consecutive leaf hashes starting with the leaf at index first.
func FetchLeafHashes(ctx context.Context, f TileFetcherFunc, first, N, logSize uint64) ([][]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.FetchLeafHashes")
	defer span.End()

	span.SetAttributes(firstKey.Int64(otel.Clamp64(first)), NKey.Int64(otel.Clamp64(N)), logSizeKey.Int64(otel.Clamp64(logSize)))

	nc := newNodeCache(f, logSize)
	hashes := make([][]byte, 0, N)
	for i, end := first, first+N; i < end; i++ {
		nID := compact.NodeID{Level: 0, Index: i}
		h, err := nc.GetNode(ctx, nID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch node %v: %v", nID, err)
		}
		hashes = append(hashes, h)
	}
	return hashes, nil
}

// GetEntryBundle fetches the entry bundle at the given _tile index_.
func GetEntryBundle(ctx context.Context, f EntryBundleFetcherFunc, i, logSize uint64) (api.EntryBundle, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.GetEntryBundle")
	defer span.End()

	span.SetAttributes(indexKey.Int64(otel.Clamp64(i)), logSizeKey.Int64(otel.Clamp64(logSize)))
	bundle := api.EntryBundle{}
	p := layout.PartialTileSize(0, i, logSize)
	sRaw, err := f(ctx, i, p)
	if err != nil {
		return bundle, fmt.Errorf("failed to fetch bundle at index %d: %v", i, err)
	}
	if err := bundle.UnmarshalText(sRaw); err != nil {
		return bundle, fmt.Errorf("failed to parse EntryBundle at index %d: %v", i, err)
	}
	return bundle, nil
}

// ProofBuilder knows how to build inclusion and consistency proofs from tiles.
// Since the tiles commit only to immutable nodes, the job of building proofs is slightly
// more complex as proofs can touch "ephemeral" nodes, so these need to be synthesized.
// This object constructs a cache internally to make it efficient for multiple operations
// at a given tree size.
type ProofBuilder struct {
	treeSize  uint64
	nodeCache nodeCache
}

// NewProofBuilder creates a new ProofBuilder object for a given tree size.
// The returned ProofBuilder can be re-used for proofs related to a given tree size, but
// it is not thread-safe and should not be accessed concurrently.
func NewProofBuilder(ctx context.Context, treeSize uint64, f TileFetcherFunc) (*ProofBuilder, error) {
	pb := &ProofBuilder{
		treeSize:  treeSize,
		nodeCache: newNodeCache(f, treeSize),
	}
	return pb, nil
}

// InclusionProof constructs an inclusion proof for the leaf at index in a tree of
// the given size.
func (pb *ProofBuilder) InclusionProof(ctx context.Context, index uint64) ([][]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.InclusionProof")
	defer span.End()

	span.SetAttributes(indexKey.Int64(otel.Clamp64(index)))

	nodes, err := proof.Inclusion(index, pb.treeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate inclusion proof node list: %v", err)
	}
	return pb.fetchNodes(ctx, nodes)
}

// ConsistencyProof constructs a consistency proof between the provided tree sizes.
func (pb *ProofBuilder) ConsistencyProof(ctx context.Context, smaller, larger uint64) ([][]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.ConsistencyProof")
	defer span.End()
	span.SetAttributes(smallerKey.Int64(otel.Clamp64(smaller)), largerKey.Int64(otel.Clamp64(larger)))

	if m := max(smaller, larger); m > pb.treeSize {
		return nil, fmt.Errorf("requested consistency proof to %d which is larger than tree size %d", m, pb.treeSize)
	}

	nodes, err := proof.Consistency(smaller, larger)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate consistency proof node list: %v", err)
	}
	return pb.fetchNodes(ctx, nodes)
}

// fetchNodes retrieves the specified proof nodes via pb's nodeCache.
func (pb *ProofBuilder) fetchNodes(ctx context.Context, nodes proof.Nodes) ([][]byte, error) {
	hashes := make([][]byte, 0, len(nodes.IDs))
	// TODO(al) parallelise this.
	for _, id := range nodes.IDs {
		h, err := pb.nodeCache.GetNode(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get node (%v): %v", id, err)
		}
		hashes = append(hashes, h)
	}
	var err error
	if hashes, err = nodes.Rehash(hashes, hasher.HashChildren); err != nil {
		return nil, fmt.Errorf("failed to rehash proof: %v", err)
	}
	return hashes, nil
}

// LogStateTracker represents a client-side view of a target log's state.
// This tracker handles verification that updates to the tracked log state are
// consistent with previously seen states.
type LogStateTracker struct {
	origin              string
	consensusCheckpoint ConsensusCheckpointFunc
	cpSigVerifier       note.Verifier
	tileFetcher         TileFetcherFunc

	// The fields under here will all be updated at the same time.
	// Access to any of these fields is guarded by mu.
	mu sync.RWMutex

	// latestConsistent is the deserialised form of LatestConsistentRaw
	latestConsistent log.Checkpoint
	// latestConsistentRaw holds the raw bytes of the latest proven-consistent
	// LogState seen by this tracker.
	latestConsistentRaw []byte
	// proofBuilder for building proofs at LatestConsistent checkpoint.
	proofBuilder *ProofBuilder
}

// NewLogStateTracker creates a newly initialised tracker.
// If a serialised LogState representation is provided then this is used as the
// initial tracked state, otherwise a log state is fetched from the target log.
func NewLogStateTracker(ctx context.Context, tF TileFetcherFunc, checkpointRaw []byte, nV note.Verifier, origin string, cc ConsensusCheckpointFunc) (*LogStateTracker, error) {
	ret := &LogStateTracker{
		origin:              origin,
		consensusCheckpoint: cc,
		cpSigVerifier:       nV,
		tileFetcher:         tF,
	}
	if len(checkpointRaw) > 0 {
		ret.latestConsistentRaw = checkpointRaw
		cp, _, _, err := log.ParseCheckpoint(checkpointRaw, origin, nV)
		if err != nil {
			return ret, err
		}
		ret.latestConsistent = *cp
		ret.proofBuilder, err = NewProofBuilder(ctx, ret.latestConsistent.Size, ret.tileFetcher)
		if err != nil {
			return ret, fmt.Errorf("NewProofBuilder: %v", err)
		}
		return ret, nil
	}
	_, _, _, err := ret.Update(ctx)
	return ret, err
}

// Update attempts to update the local view of the target log's state.
// If a more recent logstate is found, this method will attempt to prove
// that it is consistent with the local state before updating the tracker's
// view.
// Returns the old checkpoint, consistency proof, and newer checkpoint used to update.
// If the LatestConsistent checkpoint is 0 sized, no consistency proof will be returned
// since it would be meaningless to do so.
func (lst *LogStateTracker) Update(ctx context.Context) ([]byte, [][]byte, []byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.logstatetracker.Update")
	defer span.End()

	c, cRaw, _, err := lst.consensusCheckpoint(ctx, lst.cpSigVerifier, lst.origin)
	if err != nil {
		return nil, nil, nil, err
	}
	builder, err := NewProofBuilder(ctx, c.Size, lst.tileFetcher)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create proof builder: %v", err)
	}
	lst.mu.Lock()
	defer lst.mu.Unlock()
	var p [][]byte
	if lst.latestConsistent.Size > 0 {
		if c.Size <= lst.latestConsistent.Size {
			return lst.latestConsistentRaw, p, lst.latestConsistentRaw, nil
		}
		p, err = builder.ConsistencyProof(ctx, lst.latestConsistent.Size, c.Size)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := proof.VerifyConsistency(hasher, lst.latestConsistent.Size, c.Size, p, lst.latestConsistent.Hash, c.Hash); err != nil {
			return nil, nil, nil, ErrInconsistency{
				SmallerRaw: lst.latestConsistentRaw,
				LargerRaw:  cRaw,
				Proof:      p,
				Wrapped:    err,
			}
		}
		// Update is consistent,

	}
	oldRaw := lst.latestConsistentRaw
	lst.latestConsistentRaw, lst.latestConsistent = cRaw, *c
	lst.proofBuilder = builder
	return oldRaw, p, lst.latestConsistentRaw, nil
}

func (lst *LogStateTracker) Latest() log.Checkpoint {
	lst.mu.RLock()
	defer lst.mu.RUnlock()
	return lst.latestConsistent
}

// tileKey is used as a key in nodeCache's tile map.
type tileKey struct {
	tileLevel uint64
	tileIndex uint64
}

// nodeCache hides the tiles abstraction away, and improves
// performance by caching tiles it's seen.
// Not threadsafe, and intended to be only used throughout the course
// of a single request.
type nodeCache struct {
	logSize   uint64
	ephemeral map[compact.NodeID][]byte
	tiles     map[tileKey]api.HashTile
	getTile   TileFetcherFunc
}

// newNodeCache creates a new nodeCache instance for a given log size.
func newNodeCache(f TileFetcherFunc, logSize uint64) nodeCache {
	return nodeCache{
		logSize:   logSize,
		ephemeral: make(map[compact.NodeID][]byte),
		tiles:     make(map[tileKey]api.HashTile),
		getTile:   f,
	}
}

// SetEphemeralNode stored a derived "ephemeral" tree node.
func (n *nodeCache) SetEphemeralNode(id compact.NodeID, h []byte) {
	n.ephemeral[id] = h
}

// GetNode returns the internal log tree node hash for the specified node ID.
// A previously set ephemeral node will be returned if id matches, otherwise
// the tile containing the requested node will be fetched and cached, and the
// node hash returned.
func (n *nodeCache) GetNode(ctx context.Context, id compact.NodeID) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.client.nodecache.GetNode")
	defer span.End()

	span.SetAttributes(indexKey.Int64(otel.Clamp64(id.Index)), levelKey.Int64(int64(id.Level)))

	// First check for ephemeral nodes:
	if e := n.ephemeral[id]; len(e) != 0 {
		return e, nil
	}
	// Otherwise look in fetched tiles:
	tileLevel, tileIndex, nodeLevel, nodeIndex := layout.NodeCoordsToTileAddress(uint64(id.Level), uint64(id.Index))
	tKey := tileKey{tileLevel, tileIndex}
	t, ok := n.tiles[tKey]
	if !ok {
		span.AddEvent("cache miss")
		p := layout.PartialTileSize(tileLevel, tileIndex, n.logSize)
		tileRaw, err := n.getTile(ctx, tileLevel, tileIndex, p)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tile: %v", err)
		}
		var tile api.HashTile
		if err := tile.UnmarshalText(tileRaw); err != nil {
			return nil, fmt.Errorf("failed to parse tile: %v", err)
		}
		t = tile
		n.tiles[tKey] = tile
	}
	// We've got the tile, now we need to look up (or calculate) the node inside of it
	numLeaves := 1 << nodeLevel
	firstLeaf := int(nodeIndex) * numLeaves
	lastLeaf := firstLeaf + numLeaves
	if lastLeaf > len(t.Nodes) {
		return nil, fmt.Errorf("require leaf nodes [%d, %d) but only got %d leaves", firstLeaf, lastLeaf, len(t.Nodes))
	}
	rf := compact.RangeFactory{Hash: hasher.HashChildren}
	r := rf.NewEmptyRange(0)
	for _, l := range t.Nodes[firstLeaf:lastLeaf] {
		if err := r.Append(l, nil); err != nil {
			return nil, fmt.Errorf("failed to Append: %v", err)
		}
	}
	return r.GetRootHash(nil)
}
