// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Nullable wraps a value that is BOTH optional and JSON-nullable, distinguishing
// the three wire states the LSP specification assigns distinct meaning to:
//
//   - absent  — the zero Nullable; omitted on marshal via the ",omitzero" tag;
//   - null    — an explicit JSON null;
//   - value   — a present value.
//
// It is generated only for properties that are simultaneously optional and have
// a null arm (e.g. WorkspaceFoldersInitializeParams.workspaceFolders, where
// absent means "no workspace-folder support" and null means "supported, none
// open"). A plain pointer cannot represent all three states.
type Nullable[T any] struct {
	value T
	set   bool
	null  bool
}

// IsZero reports whether the value is absent. It drives the ",omitzero" tag so
// an unset Nullable is omitted entirely.
func (n Nullable[T]) IsZero() bool { return !n.set }

// IsNull reports whether the value is present as an explicit JSON null.
func (n Nullable[T]) IsNull() bool { return n.set && n.null }

// Get returns the wrapped value and whether a non-null value is present.
func (n Nullable[T]) Get() (T, bool) { return n.value, n.set && !n.null }

// MarshalJSONTo implements json.MarshalerTo, streaming the wrapped value (or
// null) through enc so encoder options propagate without materializing
// intermediate bytes per field.
func (n Nullable[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if n.null {
		return enc.WriteToken(jsontext.Null)
	}
	return json.MarshalEncode(enc, &n.value)
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It decodes the value in
// place on the caller's decoder — re-applying the union unmarshalers so a
// nested union value dispatches even when the decoder was not built by
// [Unmarshal] — instead of materializing the value bytes and re-parsing them
// with a fresh decoder per field.
func (n *Nullable[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	n.set = true
	if dec.PeekKind() == 'n' {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		n.null = true
		var zero T
		n.value = zero
		return nil
	}
	n.null = false
	return json.UnmarshalDecode(dec, &n.value, json.WithUnmarshalers(unionUnmarshalers))
}
