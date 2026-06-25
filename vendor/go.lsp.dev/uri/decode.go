// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import (
	"slices"
	"strings"
	"unicode/utf8"
)

func decodeComponent(s string) string {
	if strings.IndexByte(s, '%') < 0 {
		return s
	}
	return percentDecode(s)
}

func percentDecode(s string) string {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '%' && isAlnum(s[i+1]) && isAlnum(s[i+2]) {
			return percentDecodeFrom(s, i)
		}
	}
	return s
}

func percentDecodeFrom(s string, first int) string {
	out := make([]byte, 0, len(s))
	out = append(out, s[:first]...)
	for i := first; i < len(s); {
		if i+2 < len(s) && s[i] == '%' && isAlnum(s[i+1]) && isAlnum(s[i+2]) {
			start := i
			i += 3
			for i+2 < len(s) && s[i] == '%' && isAlnum(s[i+1]) && isAlnum(s[i+2]) {
				i += 3
			}
			out = append(out, decodePercentRunGraceful(s[start:i])...)
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}

func decodePercentRunGraceful(run string) string {
	triplets := len(run) / 3
	buf := make([]byte, triplets)
	hexOK := make([]bool, triplets)
	for i, j := 0, 0; i < len(run); i, j = i+3, j+1 {
		hi := hexDecodeTable[run[i+1]]
		lo := hexDecodeTable[run[i+2]]
		if hi|lo < 0x80 {
			buf[j] = hi<<4 | lo
			hexOK[j] = true
		}
	}

	if allTrue(hexOK) && utf8.Valid(buf) {
		return string(buf)
	}

	firstValid := firstValidUTF8HexSuffix(buf, hexOK)
	if firstValid < 0 {
		return run
	}
	return run[:firstValid*3] + string(buf[firstValid:])
}

func allTrue(values []bool) bool {
	for _, ok := range values {
		if !ok {
			return false
		}
	}
	return true
}

func firstValidUTF8HexSuffix(buf []byte, hexOK []bool) int {
	var validStart [5]bool
	validStart[len(buf)%len(validStart)] = true
	hexSuffixOK := true
	firstValid := -1

	for i := range slices.Backward(buf) {
		hexSuffixOK = hexOK[i] && hexSuffixOK
		ok := false
		if hexSuffixOK {
			r, size := utf8.DecodeRune(buf[i:])
			ok = (r != utf8.RuneError || size != 1) && validStart[(i+size)%len(validStart)]
		}
		validStart[i%len(validStart)] = ok
		if ok {
			firstValid = i
		}
	}
	return firstValid
}

func isAlnum(b byte) bool {
	return uriCharClass[b]&charClassAlnum != 0
}
