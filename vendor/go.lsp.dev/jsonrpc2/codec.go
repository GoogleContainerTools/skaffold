// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	stdjson "encoding/json"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Codec marshals and unmarshals the user payloads of a JSON-RPC message: the
// "params" of a request and the "result" of a response. The envelope itself is
// never routed through a Codec; only the user-controlled payload is.
//
// Implementations must be safe for concurrent use, must not escape HTML-sensitive
// characters (to match a json.Encoder with SetEscapeHTML(false)), and must treat
// a [RawMessage] (or encoding/json.RawMessage) as an opaque, already-encoded value
// that is passed through verbatim without re-encoding.
type Codec interface {
	// Marshal encodes v into its JSON representation. A nil v encodes to the JSON
	// null literal. A [RawMessage] value is returned verbatim.
	Marshal(v any) ([]byte, error)

	// Unmarshal decodes the JSON in data into the value pointed to by v. When v is
	// a *RawMessage the bytes are copied verbatim into a slice the caller owns.
	Unmarshal(data []byte, v any) error
}

// DefaultCodec is the [Codec] used for payloads when a connection is not given an
// explicit codec. It is backed by the encoding/json/v2 experiment
// (github.com/go-json-experiment/json) and is overridable by assigning a
// different [Codec]; the opt-in codec/sonic and codec/goccy packages provide
// drop-in alternatives.
var DefaultCodec Codec = JSONCodec{}

// jsonV2Options is the fixed option set for the default codec. HTML-sensitive
// characters are left unescaped so that payloads match the byte output of a
// json.Encoder configured with SetEscapeHTML(false); json/v2 already defaults to
// not escaping, but the option is set explicitly to make the contract clear and
// stable against future default changes.
var jsonV2Options = json.JoinOptions(jsontext.EscapeForHTML(false))

// JSONCodec is the default [Codec] implementation, backed by encoding/json/v2
// (github.com/go-json-experiment/json). It produces compact output with no
// trailing newline and no HTML escaping, consistent with json.Marshal, and
// passes [RawMessage] values through verbatim.
type JSONCodec struct{}

// compile-time check that JSONCodec satisfies Codec.
var _ Codec = JSONCodec{}

// Marshal implements [Codec] using encoding/json/v2. A [RawMessage] or
// encoding/json.RawMessage value is returned verbatim (with nil mapped to the
// null literal); every other value is encoded through json/v2.
func (JSONCodec) Marshal(v any) ([]byte, error) {
	if raw, ok := rawBytes(v); ok {
		return rawMarshal(raw), nil
	}
	return json.Marshal(v, jsonV2Options)
}

// Unmarshal implements [Codec] using encoding/json/v2. When v is a *RawMessage or
// *encoding/json.RawMessage the bytes are copied verbatim; otherwise the value is
// decoded through json/v2.
func (JSONCodec) Unmarshal(data []byte, v any) error {
	if rawUnmarshal(data, v) {
		return nil
	}
	return json.Unmarshal(data, v, jsonV2Options)
}

// rawBytes reports whether v is one of the raw, already-encoded JSON types and,
// if so, returns its bytes. Both this package's [RawMessage] and
// encoding/json.RawMessage are recognized so that either may be passed through
// verbatim.
func rawBytes(v any) (raw []byte, ok bool) {
	switch m := v.(type) {
	case RawMessage:
		return m, true
	case stdjson.RawMessage:
		return m, true
	default:
		return nil, false
	}
}

// rawMarshal returns the verbatim encoding of a raw JSON value: the bytes as-is
// when present, or the null literal when nil, matching the convention that a nil
// raw value encodes to JSON null.
func rawMarshal(raw []byte) []byte {
	if raw == nil {
		return []byte("null")
	}
	return raw
}

// rawUnmarshal copies data verbatim into v when v is a pointer to one of the raw
// JSON types, and reports whether it handled v. A nil data slice yields a nil
// destination so that "absent" stays distinguishable from "present but empty".
func rawUnmarshal(data []byte, v any) (handled bool) {
	switch p := v.(type) {
	case *RawMessage:
		*p = cloneBytes(data)
		return true
	case *stdjson.RawMessage:
		*p = cloneBytes(data)
		return true
	default:
		return false
	}
}

// marshalParams encodes the request parameters v with codec c into a
// [RawMessage] suitable for the wire envelope. A nil or empty payload yields a
// nil RawMessage so that the encoder omits the "params" member entirely (the
// convention that null parameters mean "no parameters"); a [RawMessage] is
// passed through verbatim. When c is nil the [DefaultCodec] is used.
func marshalParams(c Codec, v any) (RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	if c == nil {
		c = DefaultCodec
	}
	b, err := c.Marshal(v)
	if err != nil {
		return nil, err
	}
	// An explicit null payload is treated as "no parameters" so it is omitted
	// from the envelope, consistent with the decode-side classification.
	if len(b) == 0 || isNullLiteral(b) {
		return nil, nil
	}
	return RawMessage(b), nil
}

// unmarshalResult decodes the response result data with codec c into v. An empty
// or null result is a no-op so that a success response carrying no payload leaves
// v at its zero value rather than failing. A *RawMessage destination receives a
// verbatim copy of data. When c is nil the [DefaultCodec] is used.
func unmarshalResult(c Codec, data RawMessage, v any) error {
	if v == nil {
		return nil
	}
	if len(data) == 0 || isNullLiteral(data) {
		// Still honor a RawMessage destination so the caller can observe an
		// explicit null or empty result distinctly.
		if rawUnmarshal(data, v) {
			return nil
		}
		return nil
	}
	if c == nil {
		c = DefaultCodec
	}
	return c.Unmarshal(data, v)
}
