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

package storage

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/internal/otel"
	"golang.org/x/exp/maps"
	"k8s.io/klog/v2"
)

// SequencedEntry represents a log entry which has already been sequenced.
type SequencedEntry struct {
	// BundleData is the entry's data serialised into the correct format for appending to an entry bundle.
	BundleData []byte
	// LeafHash is the entry's Merkle leaf hash.
	LeafHash []byte
}

func Integrate(ctx context.Context, getTiles func(ctx context.Context, tileIDs []TileID, treeSize uint64) ([]*api.HashTile, error), fromSize uint64, leafHashes [][]byte) (newSize uint64, rootHash []byte, tiles map[TileID]*api.HashTile, err error) {
	tb := newTreeBuilder(getTiles)
	return tb.integrate(ctx, fromSize, leafHashes)
}

// getPopulatedTileFunc is the signature of a function which can return a fully populated tile for the given tile coords.
type getPopulatedTileFunc func(ctx context.Context, tileID TileID, treeSize uint64) (*populatedTile, error)

// treeBuilder constructs Merkle trees.
//
// This struct it indended to be used by storage implementations during the integration of entries into the log.
// treeBuilder caches data from tiles to speed things up, but has no mechanism for evicting from its internal cache,
// so while it _may_ be possible to use the same instance across a number of integration runs (e.g. if the same job
// is responsible for integrating entries for a number of contiguous trees), the lifetime should be bounded so as not
// to leak memory.
type treeBuilder struct {
	readCache *tileReadCache
	rf        *compact.RangeFactory
}

// newTreeBuilder creates a new instance of treeBuilder.
//
// The getTiles param must know how to fetch the specified tiles from storage. It must return tiles in the same order as the
// provided tileIDs, substituing nil for any tiles which were not found.
func newTreeBuilder(getTiles func(ctx context.Context, tileIDs []TileID, treeSize uint64) ([]*api.HashTile, error)) *treeBuilder {
	readCache := newTileReadCache(getTiles)
	r := &treeBuilder{
		readCache: &readCache,
		rf:        &compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren},
	}

	return r
}

// newRange creates a new compact.Range for the specified treeSize, fetching tiles as necessary.
func (t *treeBuilder) newRange(ctx context.Context, treeSize uint64) (*compact.Range, error) {
	rangeNodes := compact.RangeNodes(0, treeSize, nil)
	toFetch := make(map[TileID]struct{})
	for _, id := range rangeNodes {
		tLevel, tIndex, _, _ := layout.NodeCoordsToTileAddress(uint64(id.Level), id.Index)
		toFetch[TileID{Level: tLevel, Index: tIndex}] = struct{}{}
	}
	if err := t.readCache.Prewarm(ctx, maps.Keys(toFetch), treeSize); err != nil {
		return nil, fmt.Errorf("Prewarm: %v", err)
	}

	hashes := make([][]byte, 0, len(rangeNodes))
	for _, id := range rangeNodes {
		tLevel, tIndex, nLevel, nIndex := layout.NodeCoordsToTileAddress(uint64(id.Level), id.Index)
		ft, err := t.readCache.Get(ctx, TileID{Level: tLevel, Index: tIndex}, treeSize)
		if err != nil {
			return nil, err
		}
		h := ft.Get(compact.NodeID{Level: nLevel, Index: nIndex})
		if h == nil {
			return nil, fmt.Errorf("missing node: [%d/%d@%d]", id.Level, id.Index, treeSize)
		}
		hashes = append(hashes, h)
	}
	return t.rf.NewRange(0, treeSize, hashes)
}

func (t *treeBuilder) integrate(ctx context.Context, fromSize uint64, leafHashes [][]byte) (newSize uint64, rootHash []byte, tiles map[TileID]*api.HashTile, err error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.integrate")
	defer span.End()

	span.SetAttributes(fromSizeKey.Int64(otel.Clamp64(fromSize)), numEntriesKey.Int(len(leafHashes)))

	baseRange, err := t.newRange(ctx, fromSize)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to create range covering existing log: %w", err)
	}

	// Initialise a compact range representation, and verify the stored state.
	r, err := baseRange.GetRootHash(nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("invalid log state, unable to recalculate root: %w", err)
	}
	if len(leafHashes) == 0 {
		klog.V(1).Infof("Nothing to do.")
		// C2SP.org/log-tiles says all Merkle operations are those from RFC6962, we need to override
		// the root of the empty tree to match (compact.Range will return an empty slice).
		if fromSize == 0 {
			r = rfc6962.DefaultHasher.EmptyRoot()
		}
		// Nothing to do, nothing done.
		return fromSize, r, nil, nil
	}

	span.AddEvent("Loaded state")
	klog.V(1).Infof("Loaded state with roothash %x", r)
	// Create a new compact range which represents the update to the tree
	newRange := t.rf.NewEmptyRange(fromSize)
	tc := newTileWriteCache(fromSize, t.readCache.Get)
	visitor := tc.Visitor(ctx)
	for _, e := range leafHashes {
		// Update range and set nodes
		if err := newRange.Append(e, visitor); err != nil {
			return 0, nil, nil, fmt.Errorf("newRange.Append(): %v", err)
		}

	}
	// Check whether the visitor had any problems building the update range
	if err := tc.Err(); err != nil {
		return 0, nil, nil, err
	}
	span.AddEvent("Updated tile cache")

	// Merge the update range into the old tree
	if err := baseRange.AppendRange(newRange, visitor); err != nil {
		return 0, nil, nil, fmt.Errorf("failed to merge new range onto existing log: %w", err)
	}

	// Check whether the visitor had any problems when merging the new range into the tree
	if err := tc.Err(); err != nil {
		return 0, nil, nil, err
	}

	// Calculate the new root hash - don't pass in the tileCache visitor here since
	// this will construct any ephemeral nodes and we do not want to store those.
	newRoot, err := baseRange.GetRootHash(nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to calculate new root hash: %w", err)
	}

	span.AddEvent("Calculated new root")

	// All calculation is now complete, all that remains is to store the new
	// tiles and updated log state.
	klog.V(1).Infof("New log state: size 0x%x hash: %x", baseRange.End(), newRoot)

	return baseRange.End(), newRoot, tc.Tiles(), nil

}

// tileReadCache is a structure which provides a very simple thread-safe read-through cache based on a map of tiles.
type tileReadCache struct {
	entries  map[string]*populatedTile
	getTiles func(ctx context.Context, tileIDs []TileID, treeSize uint64) ([]*api.HashTile, error)
}

func newTileReadCache(getTiles func(ctx context.Context, tileIDs []TileID, treeSize uint64) ([]*api.HashTile, error)) tileReadCache {
	return tileReadCache{
		entries:  make(map[string]*populatedTile),
		getTiles: getTiles,
	}
}

// Get returns a previously set tile and true, or, if no such tile is in the cache, attempt to fetch it.
func (r *tileReadCache) Get(ctx context.Context, tileID TileID, treeSize uint64) (*populatedTile, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.readCache.Get")
	defer span.End()

	span.SetAttributes(indexKey.Int64(otel.Clamp64(tileID.Index)), levelKey.Int64(otel.Clamp64(tileID.Level)), treeSizeKey.Int64(otel.Clamp64(treeSize)))

	k := layout.TilePath(uint64(tileID.Level), tileID.Index, layout.PartialTileSize(tileID.Level, tileID.Index, treeSize))
	e, ok := r.entries[k]
	if !ok {
		klog.V(1).Infof("Readcache miss: %q", k)
		span.AddEvent(fmt.Sprintf("Cache miss %q", k))
		t, err := r.getTiles(ctx, []TileID{tileID}, treeSize)
		if err != nil {
			return nil, err
		}
		e, err = newPopulatedTile(t[0])
		if err != nil {
			return nil, fmt.Errorf("failed to create fulltile: %v", err)
		}
		r.entries[k] = e
	}
	return e, nil
}

// Preward fills the cache by fetching the given tilesIDs.
//
// Returns an error if any of the tiles couldn't be fetched.
func (r *tileReadCache) Prewarm(ctx context.Context, tileIDs []TileID, treeSize uint64) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.readCache.Prewarm")
	defer span.End()

	t, err := r.getTiles(ctx, tileIDs, treeSize)
	if err != nil {
		return err
	}
	for i, tile := range t {
		e, err := newPopulatedTile(tile)
		if err != nil {
			return fmt.Errorf("failed to create fulltile: %v", err)
		}
		k := layout.TilePath(uint64(tileIDs[i].Level), tileIDs[i].Index, layout.PartialTileSize(tileIDs[i].Level, tileIDs[i].Index, treeSize))
		r.entries[k] = e
	}
	return nil
}

// tileWriteCache is a simple cache for storing the newly created tiles produced by
// the integration of new leaves into the tree.
//
// Calls to Visit will cause the map of tiles to become filled with the set of
// `dirty` tiles which need to be flushed back to storage to preserve the updated
// tree state.
//
// Note that by itself, this cache does not update any persisted state.
type tileWriteCache struct {
	m   map[TileID]*populatedTile
	err []error

	treeSize uint64
	getTile  getPopulatedTileFunc
}

// newtileWriteCache creates a new cache for the given treeSize, and uses the provided
// function to fetch existing tiles which are being updated by the Visitor func.
func newTileWriteCache(treeSize uint64, getTile getPopulatedTileFunc) *tileWriteCache {
	return &tileWriteCache{
		m:        make(map[TileID]*populatedTile),
		treeSize: treeSize,
		getTile:  getTile,
	}
}

// Err returns an aggregated view of any errors seen by the visitor function.
//
// This can be used to check whether updates to the tile cache made by the visitor
// were made correctly. Any errors returned here are most likely to be due to
// the cache attempting to read an existing tile which is being updated.
func (tc *tileWriteCache) Err() error {
	return errors.Join(tc.err...)
}

// minImpliedTreeSize returns the smallest possible tree size implied by the existence of a tile
// with the given ID.
func minImpliedTreeSize(id TileID) uint64 {
	return (id.Index * layout.TileWidth) << (id.Level * 8)
}

// Visitor returns a function suitable for use with the compact.Range visitor pattern.
//
// The returned function is expected to be called sequentially to set one or nodes
// to their corresponding hash values.
func (tc *tileWriteCache) Visitor(ctx context.Context) compact.VisitFn {
	return func(id compact.NodeID, hash []byte) {
		tileLevel, tileIndex, nodeLevel, nodeIndex := layout.NodeCoordsToTileAddress(uint64(id.Level), uint64(id.Index))
		tileID := TileID{Level: tileLevel, Index: tileIndex}
		tile := tc.m[tileID]
		if tile == nil {
			var err error
			// If this tile implies a larger tree size than we started integrating at, we don't
			// need to try to fetch the tile since it probably doesn't exist.
			// If it _does_ exist, e.g. due to an earlier crash during integration, we'll discover
			// any non-idempotency issues when we come to flush these new tiles out.
			if iSize := minImpliedTreeSize(tileID); iSize <= tc.treeSize {
				tile, err = tc.getTile(ctx, tileID, tc.treeSize)
				if err != nil {
					tc.err = append(tc.err, err)
					return
				}
			}
			if tile == nil {
				// No tile found in storage: this is a brand new tile being created due to tree growth.
				tile, err = newPopulatedTile(nil)
				if err != nil {
					tc.err = append(tc.err, err)
					return
				}
			}
		}
		tc.m[tileID] = tile
		// Update the tile with the new node hash.
		idx := compact.NodeID{Level: nodeLevel, Index: nodeIndex}
		tile.Set(idx, hash)
	}
}

// Tiles returns all visited tiles.
func (tc *tileWriteCache) Tiles() map[TileID]*api.HashTile {
	newTiles := make(map[TileID]*api.HashTile)
	for k, t := range tc.m {
		newTiles[k] = &api.HashTile{Nodes: t.leaves}
	}
	return newTiles
}

// populatedTile represents a "fully populated" tile, i.e. it has all non-ephemeral internal nodes
// implied by the leaves.
type populatedTile struct {
	inner  map[compact.NodeID][]byte
	leaves [][]byte
}

// newPopulatedTile creates and populates a fullTile struct based on the passed in HashTile data.
func newPopulatedTile(h *api.HashTile) (*populatedTile, error) {
	ft := &populatedTile{
		inner:  make(map[compact.NodeID][]byte),
		leaves: make([][]byte, 0, layout.TileWidth),
	}

	if h != nil {
		// TODO: it might be better if we calculate (and cache) nodes in get, so we don't do more work that necessary.
		r := (&compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}).NewEmptyRange(0)
		for _, h := range h.Nodes {
			if err := r.Append(h, ft.Set); err != nil {
				return nil, fmt.Errorf("failed to append to range: %v", err)
			}
		}
	}
	return ft, nil
}

// Set allows setting of individual leaf/inner nodes.
// It's intended to be used as a visitor for compact.Range.
func (f *populatedTile) Set(id compact.NodeID, hash []byte) {
	if id.Level == 0 {
		if id.Index > 255 {
			panic(fmt.Sprintf("Weird node ID: %v", id))
		}
		if l, idx := uint64(len(f.leaves)), id.Index; idx >= l {
			f.leaves = append(f.leaves, make([][]byte, idx-l+1)...)
		}
		f.leaves[id.Index] = hash
	} else {
		f.inner[id] = hash
	}
}

// Get allows access to individual leaf/inner nodes.
func (f *populatedTile) Get(id compact.NodeID) []byte {
	if id.Level == 0 {
		if l := uint64(len(f.leaves)); id.Index >= l {
			return nil
		}
		return f.leaves[id.Index]
	}
	return f.inner[id]
}

func (f *populatedTile) Equals(other *populatedTile) bool {
	return reflect.DeepEqual(f, other)
}
