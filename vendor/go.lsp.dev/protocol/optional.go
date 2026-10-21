// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Optional wraps a non-nullable optional LSP property, preserving whether a
// zero value such as "" or false was present on the wire. A JSON null clears the
// value to match the legacy pointer representation, where null and absent both
// decoded to nil and were omitted on marshal.
type Optional[T any] struct {
	value T
	set   bool
}

// NewOptional returns an Optional holding v.
func NewOptional[T any](v T) Optional[T] { return Optional[T]{set: true, value: v} }

// IsZero reports whether the value is absent. It drives the ",omitzero" tag so
// an unset Optional is omitted entirely.
func (o Optional[T]) IsZero() bool { return !o.set }

// Get returns the wrapped value and whether it is present.
func (o Optional[T]) Get() (T, bool) { return o.value, o.set }

// Set marks the Optional present with v.
func (o *Optional[T]) Set(v T) {
	o.set = true
	o.value = v
}

// Clear marks the Optional absent and clears the stored value.
func (o *Optional[T]) Clear() {
	var zero T
	o.set = false
	o.value = zero
}

// MarshalJSONTo implements json.MarshalerTo, streaming the wrapped value (or
// null when absent) through enc so encoder options propagate without
// materializing intermediate bytes per field.
func (o Optional[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if !o.set {
		return enc.WriteToken(jsontext.Null)
	}
	return json.MarshalEncode(enc, &o.value)
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It decodes the value in
// place on the caller's decoder — re-applying the union unmarshalers so nested
// union values dispatch even when the decoder was not built by [Unmarshal] —
// instead of materializing the value bytes and re-parsing them with a fresh
// decoder per field.
func (o *Optional[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		o.Clear()
		return nil
	}
	o.set = true
	return json.UnmarshalDecode(dec, &o.value, json.WithUnmarshalers(unionUnmarshalers))
}
