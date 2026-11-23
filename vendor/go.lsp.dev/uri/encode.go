// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import "strings"

func encodeComponentFast(s string, isPath, isAuthority bool) string {
	for i := 0; i < len(s); i++ {
		if !canPassFast(s[i], isPath, isAuthority) {
			return encodeComponentFastFrom(s, i, isPath, isAuthority)
		}
	}
	return s
}

func encodeComponentFastFrom(s string, first int, isPath, isAuthority bool) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteString(s[:first])
	writeComponentFastFrom(&b, s, first, isPath, isAuthority)
	return b.String()
}

func writeComponentFast(b *strings.Builder, s string, isPath, isAuthority bool) {
	for i := 0; i < len(s); i++ {
		if !canPassFast(s[i], isPath, isAuthority) {
			b.WriteString(s[:i])
			writeComponentFastFrom(b, s, i, isPath, isAuthority)
			return
		}
	}
	b.WriteString(s)
}

func writeComponentFastFrom(b *strings.Builder, s string, first int, isPath, isAuthority bool) {
	for i := first; i < len(s); i++ {
		c := s[i]
		if canPassFast(c, isPath, isAuthority) {
			b.WriteByte(c)
			continue
		}
		writePercentByte(b, c)
	}
}

func encodeComponentMinimal(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' || s[i] == '?' {
			return encodeComponentMinimalFrom(s, i)
		}
	}
	return s
}

func encodeComponentMinimalFrom(s string, first int) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	b.WriteString(s[:first])
	writeComponentMinimalFrom(&b, s, first)
	return b.String()
}

func writeComponentMinimal(b *strings.Builder, s string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' || s[i] == '?' {
			b.WriteString(s[:i])
			writeComponentMinimalFrom(b, s, i)
			return
		}
	}
	b.WriteString(s)
}

func writeComponentMinimalFrom(b *strings.Builder, s string, first int) {
	for i := first; i < len(s); i++ {
		switch s[i] {
		case '#', '?':
			writePercentByte(b, s[i])
		default:
			b.WriteByte(s[i])
		}
	}
}

func canPassFast(c byte, isPath, isAuthority bool) bool {
	class := uriCharClass[c]
	if class&charClassUnreserved != 0 {
		return true
	}
	if isPath && class&charClassPathExtra != 0 {
		return true
	}
	if isAuthority && class&charClassAuthorityExtra != 0 {
		return true
	}
	return false
}

func writePercentByte(b *strings.Builder, c byte) {
	start := int(c) * 3
	b.WriteString(percentTriplets[start : start+3])
}
