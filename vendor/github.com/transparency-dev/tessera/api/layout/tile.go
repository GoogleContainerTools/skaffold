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

package layout

const (
	// TileHeight is the maximum number of levels Merkle tree levels a tile represents.
	// This is fixed at 8 by tlog-tile spec.
	TileHeight = 8
	// TileWidth is the maximum number of hashes which can be present in the bottom row of a tile.
	TileWidth = 1 << TileHeight
	// EntryBundleWidth is the maximum number of entries which can be present in an EntryBundle.
	// This is defined to be the same as the width of the node tiles by tlog-tile spec.
	EntryBundleWidth = TileWidth
)

// PartialTileSize returns the expected number of leaves in a tile at the given tile level and index
// within a tree of the specified logSize, or 0 if the tile is expected to be fully populated.
func PartialTileSize(level, index, logSize uint64) uint8 {
	sizeAtLevel := logSize >> (level * TileHeight)
	fullTiles := sizeAtLevel / TileWidth
	if index < fullTiles {
		return 0
	}
	return uint8(sizeAtLevel % TileWidth)
}

// NodeCoordsToTileAddress returns the (TileLevel, TileIndex) in tile-space, and the
// (NodeLevel, NodeIndex) address within that tile of the specified tree node co-ordinates.
func NodeCoordsToTileAddress(treeLevel, treeIndex uint64) (uint64, uint64, uint, uint64) {
	tileRowWidth := uint64(1 << (TileHeight - treeLevel%TileHeight))
	tileLevel := treeLevel / TileHeight
	tileIndex := treeIndex / tileRowWidth
	nodeLevel := uint(treeLevel % TileHeight)
	nodeIndex := uint64(treeIndex % tileRowWidth)

	return tileLevel, tileIndex, nodeLevel, nodeIndex
}
