// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"unicode/utf8"
	"unsafe"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type appendMarshaler interface {
	appendLSPJSON(dst []byte) (out []byte, err error)
}

// nullLiteral is the JSON null literal appended by the generated encoders.
const nullLiteral = "null"

func appendObjectName(dst []byte, first *bool, name string) []byte {
	if *first {
		*first = false
	} else {
		dst = append(dst, ',')
	}
	// Field names are generated/static ASCII identifiers, so direct quoting is
	// equivalent to json string quoting without the per-field scan.
	dst = append(dst, '"')
	dst = append(dst, name...)
	dst = append(dst, '"', ':')
	return dst
}

func appendJSONString(dst []byte, s string) []byte {
	// Fast path: a printable-ASCII run without quotes or backslashes needs no
	// escaping and no UTF-8 validation, so it appends verbatim. The unsafe
	// view is read-only and never escapes this frame. Admission evidence for
	// the SWAR scan lives in .bench/phase3-kernel-admission.md.
	if n := dvScanStringSpecial(unsafe.Slice(unsafe.StringData(s), len(s)), 0); n == len(s) {
		dst = append(dst, '"')
		dst = append(dst, s...)
		return append(dst, '"')
	}
	// jsontext.AppendQuote already replaces invalid UTF-8 with U+FFFD. Its
	// only error for string input is reporting invalid UTF-8, which is allowed
	// by wireOptions, so ignoring the error preserves this package's wire
	// contract while keeping the direct append path allocation-free.
	dst, _ = jsontext.AppendQuote(dst, s)
	return dst
}

func appendRawJSONValue(dst []byte, v LSPAny) ([]byte, error) {
	if v == nil {
		return append(dst, "null"...), nil
	}
	_, n, err := dvValue(v, 0)
	if err != nil {
		return nil, err
	}
	if err := dvEnd(v, n); err != nil {
		return nil, err
	}
	if rawValueNeedsReencode(v) {
		// The streaming oracle normalizes raw values on re-encode: it strips
		// insignificant whitespace, resolves string escapes, and mangles
		// invalid UTF-8 to U+FFFD under AllowInvalidUTF8. Verbatim append is
		// only byte-identical when none of those apply; this cold path keeps
		// parity for the rest. Real LSP data payloads are compact, unescaped,
		// valid UTF-8 and stay on the verbatim path.
		return appendJSONMarshal(dst, v)
	}
	return append(dst, v...), nil
}

// rawValueNeedsReencode reports whether a structurally valid raw JSON value
// would be rewritten by the streaming encoder: insignificant whitespace
// outside strings, any escape sequence inside strings, or invalid UTF-8.
func rawValueNeedsReencode(v []byte) bool {
	inStr := false
	sawHigh := false
	for _, c := range v {
		if inStr {
			switch {
			case c == '\\':
				return true
			case c == '"':
				inStr = false
			case c >= 0x80:
				sawHigh = true
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case ' ', '\t', '\n', '\r':
			return true
		}
	}
	return sawHigh && !utf8.Valid(v)
}

func appendJSONMarshal(dst []byte, v any) ([]byte, error) {
	b, err := json.Marshal(v, wireOptions)
	if err != nil {
		return nil, err
	}
	return append(dst, b...), nil
}

func appendInt32JSON(dst []byte, v int32) []byte {
	return strconv.AppendInt(dst, int64(v), 10)
}

func appendUint32JSON(dst []byte, v uint32) []byte {
	return appendUint32Decimal(dst, v)
}

func appendBoolJSON(dst []byte, v bool) []byte {
	if v {
		return append(dst, "true"...)
	}
	return append(dst, "false"...)
}

// appendFloat64JSON appends v formatted exactly as jsonwire.AppendFloat does
// for 64-bit floats (ECMA-262 §7.1.12.1 / RFC 8785 §3.2.2.3, except -0 keeps
// its sign), so the generated append encoders stay byte-identical to the
// streaming oracle. Non-finite values error like the jsontext encoder.
func appendFloat64JSON(dst []byte, v float64) ([]byte, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("protocol: unsupported value %v", v)
	}
	abs := math.Abs(v)
	format := byte('f')
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	dst = strconv.AppendFloat(dst, v, format, -1, 64)
	if format == 'e' {
		// Clean up e-09 to e-9, mirroring jsonwire.
		if n := len(dst); n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst, nil
}

// appendStringSliceJSON appends a string-kinded slice as a JSON array, sizing
// the reservation from the element lengths so the common case appends into one
// allocation.
func appendStringSliceJSON[T ~string](dst []byte, x []T) []byte {
	grow := 2
	for _, v := range x {
		grow += len(v) + 3
	}
	dst = slices.Grow(dst, grow)
	dst = append(dst, '[')
	for i, v := range x {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendJSONString(dst, string(v))
	}
	return append(dst, ']')
}

// appendUint32SliceJSON appends a uint32-kinded slice as a JSON array.
func appendUint32SliceJSON[T ~uint32](dst []byte, x []T) []byte {
	dst = slices.Grow(dst, 2+len(x)*4)
	dst = append(dst, '[')
	for i, v := range x {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendUint32JSON(dst, uint32(v))
	}
	return append(dst, ']')
}

// appendDiagnosticTagsJSON appends the compact DiagnosticTags representation
// as a JSON array.
func appendDiagnosticTagsJSON(dst []byte, x DiagnosticTags) []byte {
	dst = append(dst, '[')
	if x.n > 0 {
		dst = appendUint32JSON(dst, uint32(x.first))
		for _, v := range x.rest[:max(x.n-1, 0)] {
			dst = append(dst, ',')
			dst = appendUint32JSON(dst, uint32(v))
		}
	}
	return append(dst, ']')
}
