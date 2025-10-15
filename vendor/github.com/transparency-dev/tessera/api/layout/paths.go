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

// Package layout contains routines for specifying the path layout of Tessera logs,
// which is really to say that it provides functions to calculate paths used by the
// [tlog-tiles API].
//
// [tlog-tiles API]: https://c2sp.org/tlog-tiles
package layout

import (
	"fmt"
	"iter"
	"math"
	"strconv"
	"strings"
)

const (
	// CheckpointPath is the location of the file containing the log checkpoint.
	CheckpointPath = "checkpoint"
)

// EntriesPathForLogIndex builds the local path at which the leaf with the given index lives in.
// Note that this will be an entry bundle containing up to 256 entries and thus multiple
// indices can map to the same output path.
// The logSize is required so that a partial qualifier can be appended to tiles that
// would contain fewer than 256 entries.
func EntriesPathForLogIndex(seq, logSize uint64) string {
	tileIndex := seq / EntryBundleWidth
	return EntriesPath(tileIndex, PartialTileSize(0, tileIndex, logSize))
}

// Range returns an iterator over a list of RangeInfo structs which describe the bundles/tiles
// necessary to cover the specified range of individual entries/hashes `[from, min(from+N, treeSize) )`.
//
// If from >= treeSize or N == 0, the returned iterator will yield no elements.
func Range(from, N, treeSize uint64) iter.Seq[RangeInfo] {
	return func(yield func(RangeInfo) bool) {
		// Range is empty if we're entirely beyond the extent of the tree, or we've been asked for zero items.
		if from >= treeSize || N == 0 {
			return
		}
		// Truncate range at size of tree if necessary.
		if from+N > treeSize {
			N = treeSize - from
		}

		endInc := from + N - 1
		sIndex := from / EntryBundleWidth
		eIndex := endInc / EntryBundleWidth

		for idx := sIndex; idx <= eIndex; idx++ {
			ri := RangeInfo{
				Index: idx,
				N:     EntryBundleWidth,
			}

			switch ri.Index {
			case sIndex:
				ri.Partial = PartialTileSize(0, sIndex, treeSize)
				ri.First = uint(from % EntryBundleWidth)
				ri.N = uint(EntryBundleWidth) - ri.First

				// Handle corner-case where the range is entirely contained in first bundle, if applicable:
				if ri.Index == eIndex {
					ri.N = uint((endInc)%EntryBundleWidth) - ri.First + 1
				}
			case eIndex:
				ri.Partial = PartialTileSize(0, eIndex, treeSize)
				ri.N = uint((endInc)%EntryBundleWidth) + 1
			}

			if !yield(ri) {
				return
			}
		}
	}
}

// RangeInfo describes a specific range of elements within a particular bundle/tile.
//
// Usage:
//
//	bundleRaw, ... := fetchBundle(..., ri.Index, ri.Partial")
//	bundle, ... := parseBundle(bundleRaw)
//	elements := bundle.Entries[ri.First : ri.First+ri.N]
type RangeInfo struct {
	// Index is the index of the entry bundle/tile in the tree.
	Index uint64
	// Partial is the partial size of the bundle/tile, or zero if a full bundle/tile is expected.
	Partial uint8
	// First is the offset into the entries contained by the bundle/tile at which the range starts.
	First uint
	// N is the number of entries, starting at First, which are covered by the range.
	N uint
}

// NWithSuffix returns a tiles-spec "N" path, with a partial suffix if p > 0.
func NWithSuffix(l, n uint64, p uint8) string {
	suffix := ""
	if p > 0 {
		suffix = fmt.Sprintf(".p/%d", p)
	}
	return fmt.Sprintf("%s%s", fmtN(n), suffix)
}

// EntriesPath returns the local path for the nth entry bundle. p denotes the partial
// tile size, or 0 if the tile is complete.
func EntriesPath(n uint64, p uint8) string {
	return fmt.Sprintf("tile/entries/%s", NWithSuffix(0, n, p))
}

// TilePath builds the path to the subtree tile with the given level and index in tile space.
// If p > 0 the path represents a partial tile.
func TilePath(tileLevel, tileIndex uint64, p uint8) string {
	return fmt.Sprintf("tile/%d/%s", tileLevel, NWithSuffix(tileLevel, tileIndex, p))
}

// fmtN returns the "N" part of a Tiles-spec path.
//
// N is grouped into chunks of 3 decimal digits, starting with the most significant digit, and
// padding with zeroes as necessary.
// Digit groups are prefixed with "x", except for the least-significant group which has no prefix,
// and separated with slashes.
//
// See https://github.com/C2SP/C2SP/blob/main/tlog-tiles.md#:~:text=index%201234067%20will%20be%20encoded%20as%20x001/x234/067
func fmtN(N uint64) string {
	n := fmt.Sprintf("%03d", N%1000)
	N /= 1000
	for N > 0 {
		n = fmt.Sprintf("x%03d/%s", N%1000, n)
		N /= 1000
	}
	return n
}

// ParseTileLevelIndexPartial takes level and index in string, validates and returns the level, index and width in uint64.
//
// Examples:
// "/tile/0/x001/x234/067" means level 0 and index 1234067 of a full tile.
// "/tile/0/x001/x234/067.p/8" means level 0, index 1234067 and width 8 of a partial tile.
func ParseTileLevelIndexPartial(level, index string) (uint64, uint64, uint8, error) {
	l, err := ParseTileLevel(level)
	if err != nil {
		return 0, 0, 0, err
	}

	i, w, err := ParseTileIndexPartial(index)
	if err != nil {
		return 0, 0, 0, err
	}

	return l, i, w, err
}

// ParseTileLevel takes level in string, validates and returns the level in uint64.
func ParseTileLevel(level string) (uint64, error) {
	l, err := strconv.ParseUint(level, 10, 64)
	// Verify that level is an integer between 0 and 63 as specified in the tlog-tiles specification.
	if l > 63 || err != nil {
		return 0, fmt.Errorf("failed to parse tile level")
	}
	return l, err
}

// ParseTileIndexPartial takes index in string, validates and returns the index and width in uint64.
func ParseTileIndexPartial(index string) (uint64, uint8, error) {
	w := uint8(0)
	indexPaths := strings.Split(index, "/")

	if strings.Contains(index, ".p") {
		var err error
		w64, err := strconv.ParseUint(indexPaths[len(indexPaths)-1], 10, 64)
		if err != nil || w64 < 1 || w64 >= TileWidth {
			return 0, 0, fmt.Errorf("failed to parse tile width")
		}
		w = uint8(w64)
		indexPaths[len(indexPaths)-2] = strings.TrimSuffix(indexPaths[len(indexPaths)-2], ".p")
		indexPaths = indexPaths[:len(indexPaths)-1]
	}

	if strings.Count(index, "x") != len(indexPaths)-1 || strings.HasPrefix(indexPaths[len(indexPaths)-1], "x") {
		return 0, 0, fmt.Errorf("failed to parse tile index")
	}

	i := uint64(0)
	for _, indexPath := range indexPaths {
		indexPath = strings.TrimPrefix(indexPath, "x")
		n, err := strconv.ParseUint(indexPath, 10, 64)
		if err != nil || n >= 1000 || len(indexPath) != 3 {
			return 0, 0, fmt.Errorf("failed to parse tile index")
		}
		if i > (math.MaxUint64-n)/1000 {
			return 0, 0, fmt.Errorf("failed to parse tile index")
		}
		i = i*1000 + n
	}

	return i, w, nil
}
