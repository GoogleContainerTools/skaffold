// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "github.com/go-json-experiment/json/jsontext"

// LSPAny is the LSP "any" type: any valid JSON value (object, array, string,
// number, boolean or null).
//
// It is represented as a raw JSON value (jsontext.Value) rather than a sealed
// interface: this round-trips byte-for-byte without an "any"/interface{} field
// and defers typing to the caller, who may decode it on demand.
type LSPAny = jsontext.Value

// LSPObject is an LSP object: a JSON object whose values are LSPAny.
type LSPObject = map[string]jsontext.Value

// LSPArray is an LSP array: a JSON array whose elements are LSPAny.
type LSPArray = []jsontext.Value
