// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"unicode/utf8"
	"unsafe"
)

// hexDigits is the lookup table used when escaping control characters into the
// \u00XX form.
const hexDigits = "0123456789abcdef"

// needsEscape reports whether s contains any byte that must be escaped inside a
// JSON string literal. The common case for JSON-RPC method names and string
// identifiers is that no byte needs escaping, so this fast scan lets the encoder
// copy the bytes verbatim.
func needsEscape(s string) bool {
	for i := range len(s) {
		c := s[i]
		if c < 0x20 || c == '"' || c == '\\' {
			return true
		}
	}
	return false
}

// appendQuotedString appends s to dst as a quoted JSON string, escaping only the
// bytes that require it. HTML-sensitive characters (<, >, &) are not escaped, to
// match the behavior of a json.Encoder with SetEscapeHTML(false).
func appendQuotedString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	if !needsEscape(s) {
		dst = append(dst, s...)
		return append(dst, '"')
	}

	start := 0
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				i++
				continue
			}
			dst = append(dst, s[start:i]...)
			switch c {
			case '"':
				dst = append(dst, '\\', '"')
			case '\\':
				dst = append(dst, '\\', '\\')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			}
			i++
			start = i
			continue
		}
		i++
	}
	dst = append(dst, s[start:]...)
	return append(dst, '"')
}

// borrowJSONString decodes a JSON string span like [unquoteJSONString], but on
// the escape-free fast path it returns a string header aliasing the span's
// bytes instead of copying them. The returned string is valid only as long as
// the underlying buffer; callers own the lifetime contract.
func borrowJSONString(span []byte) (s string, ok bool) {
	if len(span) < 2 || span[0] != '"' || span[len(span)-1] != '"' {
		return "", false
	}
	body := span[1 : len(span)-1]
	for i := range len(body) {
		if body[i] == '\\' {
			// Escapes force the decoding copy; the result owns its bytes.
			return unquoteJSONString(span)
		}
	}
	if len(body) == 0 {
		return "", true
	}
	return unsafe.String(unsafe.SliceData(body), len(body)), true
}

// unquoteJSONString decodes a JSON string span (including the surrounding double
// quotes) into a Go string. It returns ok=false when the span is not a
// well-formed JSON string.
//
// The fast path returns the contents verbatim when no escape sequence is present.
func unquoteJSONString(span []byte) (s string, ok bool) {
	if len(span) < 2 || span[0] != '"' || span[len(span)-1] != '"' {
		return "", false
	}
	body := span[1 : len(span)-1]

	// Fast path: no escapes means the body is the string.
	esc := false
	for i := range len(body) {
		if body[i] == '\\' {
			esc = true
			break
		}
	}
	if !esc {
		return string(body), true
	}

	buf := make([]byte, 0, len(body))
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			buf = append(buf, c)
			i++
			continue
		}
		i++
		if i >= len(body) {
			return "", false
		}
		switch body[i] {
		case '"':
			buf = append(buf, '"')
		case '\\':
			buf = append(buf, '\\')
		case '/':
			buf = append(buf, '/')
		case 'b':
			buf = append(buf, '\b')
		case 'f':
			buf = append(buf, '\f')
		case 'n':
			buf = append(buf, '\n')
		case 'r':
			buf = append(buf, '\r')
		case 't':
			buf = append(buf, '\t')
		case 'u':
			r, n, okHex := decodeUnicodeEscape(body[i:])
			if !okHex {
				return "", false
			}
			buf = utf8.AppendRune(buf, r)
			i += n
			continue
		default:
			return "", false
		}
		i++
	}
	return string(buf), true
}

// decodeUnicodeEscape decodes a \uXXXX sequence (and a following low surrogate
// when the first escape is a high surrogate). The input s must begin at the 'u'
// of the escape sequence. It returns the decoded rune, the number of bytes of s
// consumed (counting from the 'u'), and whether decoding succeeded.
func decodeUnicodeEscape(s []byte) (r rune, n int, ok bool) {
	hi, okHi := readHex4(s)
	if !okHi {
		return 0, 0, false
	}
	// 'u' + 4 hex digits.
	n = 5
	if hi < 0xD800 || hi > 0xDFFF {
		return rune(hi), n, true
	}
	if hi >= 0xDC00 {
		// Unpaired low surrogate.
		return utf8.RuneError, n, true
	}
	// High surrogate; expect a following \uXXXX low surrogate.
	if len(s) < n+2 || s[n] != '\\' || s[n+1] != 'u' {
		return utf8.RuneError, n, true
	}
	lo, okLo := readHex4(s[n+1:])
	if !okLo || lo < 0xDC00 || lo > 0xDFFF {
		return utf8.RuneError, n, true
	}
	r = 0x10000 + (rune(hi)-0xD800)<<10 + (rune(lo) - 0xDC00)
	n += 6
	return r, n, true
}

// readHex4 reads exactly four hexadecimal digits that follow the 'u' at s[0]. It
// returns the decoded 16-bit value and whether four valid digits were present.
func readHex4(s []byte) (v uint16, ok bool) {
	if len(s) < 5 {
		return 0, false
	}
	for i := 1; i <= 4; i++ {
		c := s[i]
		var d uint16
		switch {
		case c >= '0' && c <= '9':
			d = uint16(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint16(c-'A') + 10
		default:
			return 0, false
		}
		v = v<<4 | d
	}
	return v, true
}
