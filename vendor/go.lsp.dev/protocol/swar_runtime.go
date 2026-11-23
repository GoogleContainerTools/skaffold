// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"encoding/binary"
	"math/bits"
)

// This file implements the shared SWAR (SIMD-within-a-register) byte
// classifiers behind the string hot paths: one primitive family answers
// "where is the next interesting byte" eight bytes per step for the decode
// scanners (dvString, dvStringEnd, scanString) and the encode quote fast path
// (appendJSONString). Pure portable Go — wider amd64 kernels would slot in
// behind the same two functions if a profile ever justifies them.
//
// Correctness of the first-index guarantee: the sub-borrow in swarHasZero and
// swarLessThan0x20 can corrupt LANES ABOVE the triggering byte (little-endian
// loads put later input bytes in higher lanes), never below it, so
// bits.TrailingZeros64 always reports the true first match even when a borrow
// has falsely flagged a later lane.

const (
	swarLo uint64 = 0x0101010101010101
	swarHi uint64 = 0x8080808080808080
)

// swarHasZero flags 0x80 in every lane whose byte is zero.
func swarHasZero(v uint64) uint64 {
	return (v - swarLo) & ^v & swarHi
}

// swarLessThan0x20 flags 0x80 in every lane whose byte is an unescaped
// control character (<0x20). Lanes ≥0x80 are never flagged (their high bit
// clears the &^v term), which is fine for both callers: the plain scanner
// does not care and the special scanner ORs in the high-bit mask itself.
func swarLessThan0x20(v uint64) uint64 {
	return (v - swarLo*0x20) & ^v & swarHi
}

// dvScanQuoteBackslash returns the index of the first '"' or '\\' at or after
// i, or len(raw) when neither occurs.
func dvScanQuoteBackslash(raw []byte, i int) int {
	for ; i+8 <= len(raw); i += 8 {
		w := binary.LittleEndian.Uint64(raw[i:])
		m := swarHasZero(w^(swarLo*'"')) | swarHasZero(w^(swarLo*'\\'))
		if m != 0 {
			return i + bits.TrailingZeros64(m)>>3
		}
	}
	for ; i < len(raw); i++ {
		if c := raw[i]; c == '"' || c == '\\' {
			break
		}
	}
	return i
}

// dvScanStringSpecial returns the index of the first byte at or after i that
// the zero-copy string fast path cannot consume verbatim: a quote, a
// backslash, a control byte, or a non-ASCII byte. Returns len(raw) when the
// run is clean to the end.
func dvScanStringSpecial(raw []byte, i int) int {
	for ; i+8 <= len(raw); i += 8 {
		w := binary.LittleEndian.Uint64(raw[i:])
		m := swarHasZero(w^(swarLo*'"')) | swarHasZero(w^(swarLo*'\\')) |
			(w & swarHi) | swarLessThan0x20(w)
		if m != 0 {
			return i + bits.TrailingZeros64(m)>>3
		}
	}
	for ; i < len(raw); i++ {
		c := raw[i]
		if c == '"' || c == '\\' || c < 0x20 || c >= 0x80 {
			break
		}
	}
	return i
}
