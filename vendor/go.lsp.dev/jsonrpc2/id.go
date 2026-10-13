// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"fmt"
	"io"
	"strconv"
)

// idKind discriminates the three states of an [ID].
type idKind uint8

const (
	// idNone is the zero value: the ID is unset (for example, a notification or a
	// parse-error response that carries a null id).
	idNone idKind = iota
	// idNumber means the ID holds an integer value.
	idNumber
	// idString means the ID holds a string value.
	idString
)

// ID is a JSON-RPC request identifier.
//
// Per the specification an identifier is a string, an integer, or null. ID
// stores the value without boxing it into an interface, so constructing and
// encoding an integer identifier performs no heap allocation.
//
// The zero value is a valid, unset identifier (kind none); it encodes as the
// JSON null literal and is used for notifications and for error responses that
// have no associated request id.
type ID struct {
	str  string
	num  int64
	kind idKind
}

// compile-time check that ID implements fmt.Formatter.
var _ fmt.Formatter = ID{}

// NewNumberID returns an [ID] holding the integer value v.
func NewNumberID(v int64) ID { return ID{num: v, kind: idNumber} }

// NewStringID returns an [ID] holding the string value v.
func NewStringID(v string) ID { return ID{str: v, kind: idString} }

// IsValid reports whether the identifier is set (a number or a string). The zero
// value reports false.
func (id ID) IsValid() bool { return id.kind != idNone }

// IsNumber reports whether the identifier holds an integer value.
func (id ID) IsNumber() bool { return id.kind == idNumber }

// IsString reports whether the identifier holds a string value.
func (id ID) IsString() bool { return id.kind == idString }

// Number returns the integer value of the identifier and whether it is a number.
func (id ID) Number() (int64, bool) { return id.num, id.kind == idNumber }

// String returns the string value of the identifier and whether it is a string.
func (id ID) StringValue() (string, bool) { return id.str, id.kind == idString }

// appendID appends the JSON encoding of the identifier to dst.
//
// A number is written with strconv.AppendInt, a string is written through the
// shared JSON string escaper, and an unset identifier is written as null. No
// reflection or interface boxing is involved.
func (id ID) appendID(dst []byte) []byte {
	switch id.kind {
	case idNumber:
		return strconv.AppendInt(dst, id.num, 10)
	case idString:
		return appendQuotedString(dst, id.str)
	default:
		return append(dst, 'n', 'u', 'l', 'l')
	}
}

// decodeID decodes an identifier from a JSON value span.
//
// The span must be a trimmed JSON value: a quoted string, an integer, or the
// null literal. It returns ok=false for any other shape (for example a
// fractional number or an object).
func decodeID(span []byte) (id ID, ok bool) {
	if len(span) == 0 {
		return ID{}, false
	}
	switch span[0] {
	case 'n':
		if isNullLiteral(span) {
			return ID{}, true
		}
		return ID{}, false
	case '"':
		s, sok := unquoteJSONString(span)
		if !sok {
			return ID{}, false
		}
		return NewStringID(s), true
	default:
		n, nok := parseInt64Bytes(span)
		if !nok {
			return ID{}, false
		}
		return NewNumberID(n), true
	}
}

// parseInt64Bytes parses a base-10 integer directly from a JSON number span,
// avoiding the string conversion that would otherwise allocate on scanner hot
// paths. The accepted syntax is the JSON-RPC identifier subset: an optional
// leading '-' followed by decimal digits only.
func parseInt64Bytes(span []byte) (int64, bool) {
	if len(span) == 0 {
		return 0, false
	}

	i := 0
	neg := false
	if span[0] == '-' {
		neg = true
		i = 1
		if i == len(span) {
			return 0, false
		}
	}

	const maxInt64 = uint64(1<<63 - 1)
	limit := maxInt64
	if neg {
		limit = maxInt64 + 1
	}

	var n uint64
	for ; i < len(span); i++ {
		c := span[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		d := uint64(c - '0')
		if n > (limit-d)/10 {
			return 0, false
		}
		n = n*10 + d
	}

	if neg {
		if n == maxInt64+1 {
			return -1 << 63, true
		}
		return -int64(n), true
	}
	return int64(n), true
}

// Format implements [fmt.Formatter].
//
// For the %q verb the representation is unambiguous: string forms are quoted and
// number forms are preceded by '#'. For every other verb a string is written as
// its text and a number as its decimal digits. An unset identifier is written as
// the literal "null".
func (id ID) Format(f fmt.State, r rune) {
	var s string
	switch id.kind {
	case idString:
		if r == 'q' {
			s = strconv.Quote(id.str)
		} else {
			s = id.str
		}
	case idNumber:
		if r == 'q' {
			s = "#" + strconv.FormatInt(id.num, 10)
		} else {
			s = strconv.FormatInt(id.num, 10)
		}
	default:
		s = "null"
	}
	// Format writes to the formatter's state; like the standard library's own
	// Formatter implementations, any write error is not actionable here.
	_, _ = io.WriteString(f, s)
}

// isNullLiteral reports whether span is exactly the JSON null literal.
func isNullLiteral(span []byte) bool {
	return len(span) == 4 && span[0] == 'n' && span[1] == 'u' && span[2] == 'l' && span[3] == 'l'
}
