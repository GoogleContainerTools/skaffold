// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "slices"

// uint32DigitPairs is the two-digit decimal lookup table ("00".."99") behind
// appendUint32Decimal.
var uint32DigitPairs = func() (t [200]byte) {
	for i := range 100 {
		t[i*2] = byte('0' + i/10)
		t[i*2+1] = byte('0' + i%10)
	}
	return t
}()

// appendUint32Decimal appends v in decimal using two digits per step, halving
// the divide count of strconv.AppendUint on the token-array hot path. The
// length is known up front, so digits are written in place right-to-left.
func appendUint32Decimal(dst []byte, v uint32) []byte {
	n := uint32DecimalLen(v)
	dst = slices.Grow(dst, n)
	dst = dst[:len(dst)+n]
	i := len(dst)
	for v >= 100 {
		q := v / 100
		r := (v - q*100) * 2
		i -= 2
		dst[i] = uint32DigitPairs[r]
		dst[i+1] = uint32DigitPairs[r+1]
		v = q
	}
	if v >= 10 {
		dst[i-2] = uint32DigitPairs[v*2]
		dst[i-1] = uint32DigitPairs[v*2+1]
	} else {
		dst[i-1] = byte('0' + v)
	}
	return dst
}

func appendUint32JSONArray(dst []byte, data []uint32) []byte {
	dst = slices.Grow(dst, uint32JSONArrayLen(data))
	dst = append(dst, '[')
	for i, v := range data {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendUint32Decimal(dst, v)
	}
	dst = append(dst, ']')
	return dst
}

func uint32JSONArrayLen(data []uint32) int {
	if len(data) == 0 {
		return len(`[]`)
	}
	n := len(`[]`) + len(data) - 1
	for _, v := range data {
		n += uint32DecimalLen(v)
	}
	return n
}

func uint32DecimalLen(v uint32) int {
	switch {
	case v < 10:
		return 1
	case v < 100:
		return 2
	case v < 1000:
		return 3
	case v < 10_000:
		return 4
	case v < 100_000:
		return 5
	case v < 1_000_000:
		return 6
	case v < 10_000_000:
		return 7
	case v < 100_000_000:
		return 8
	case v < 1_000_000_000:
		return 9
	default:
		return 10
	}
}
