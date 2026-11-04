// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package protocol contains Go types and the client/server RPC layer for the
// Language Server Protocol (LSP) version 3.18, with the types generated from the
// official LSP meta-model (metaModel.json).
//
// Union ("or") types in the protocol are represented as sealed Go interfaces:
// each arm is a distinct concrete type implementing a private marker method, so
// callers discriminate arms with a type switch. Decoding is performed by
// [Unmarshal], which dispatches each union to a discriminating decoder; encode
// with [Marshal]. The LSPAny type is a raw JSON value (jsontext.Value).
//
// The RPC layer ([NewServer], [NewClient], the [Server] and [Client] interfaces,
// and their dispatchers) runs over go.lsp.dev/jsonrpc2, with union-aware payload
// marshaling supplied through a jsonrpc2.Codec.
//
// Generated URI and URI fields use go.lsp.dev/uri.URI directly. The local
// [URI] type remains as a package-local compatibility and sealed-union bridge
// for arms such as [RelativePatternBaseURI], where Go requires a local receiver
// type for the generated marker method. Prefer go.lsp.dev/uri.URI for ordinary
// fields; convert explicitly to protocol.URI only at those union boundaries.
//
// The generator lives in go.lsp.dev/protocol/internal/genlsp.
package protocol
