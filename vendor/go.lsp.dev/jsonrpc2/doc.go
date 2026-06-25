// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

// Package jsonrpc2 is a minimal, allocation-conscious implementation of the
// JSON-RPC 2.0 wire protocol.
//
// The package is built around a reflection-free wire core plus pluggable
// framing, a swappable payload codec, and a bidirectional connection state
// machine. The wire core encodes message envelopes by appending directly into a
// byte buffer ([EncodeMessage], [AppendMessage], [AppendCall],
// [AppendNotification], [AppendResponse], [AppendBatch]) and decodes them with
// a single-pass span scanner ([DecodeMessage], [ParseRequests]), so the hot
// path performs no reflection and no payload copies: a served request borrows
// its method and params straight from the transport frame, the request
// bookkeeping is pooled, and dispatch is direct-return, which is what lets a
// void round trip measure zero allocations.
//
// The borrow has one rule: a [Handler]'s request, its Method, and its Params
// are valid only until the handler returns. [Request.Clone] takes an owned
// copy, [Async] clones automatically when a handler detaches, and
// [DetachContext] keeps a context alive past the handler. The same contract
// applies to requests decoded by [DecodeMessage] and [ParseRequests], whose
// results borrow their input; responses and error members are owned. For
// callback-scoped parser fast paths, [ScanMessageView], [ScanFrameView], and
// [AppendRequestViews] expose the same kind of borrowed views over
// caller-owned frame bytes.
//
// Runtime modes are explicit: [Conn]/[Peer] is bidirectional, [SingleClient]
// serializes calls with a caller-owned read loop, [PipelineClient] keeps
// concurrent client-originated calls in flight without dispatching
// server-initiated requests, and [BatchClient] exposes raw-frame batch I/O.
// [NewChannelStreamPair] supplies an in-memory encoded-frame transport for
// same-process peers that still want full JSON-RPC wire encoding and scanning.
//
// The wire-model message types are a closed set of [*Call], [*Notification],
// and [*Response], all of which implement the [Message] interface; handlers
// receive the concrete [Request] instead.
//
// # Framing
//
// A [Stream] adapts a byte transport to message reads and writes. Two framings
// are provided: newline-delimited JSON ([NewNDJSONStream], compatible with the
// Model Context Protocol stdio transport) and LSP "Content-Length" header
// framing ([NewHeaderStream]). [NewStream] selects the header framing as the
// gopls-compatible default.
//
// # Codec
//
// The envelope is never marshaled through a codec; only the user payload
// (params and result) is. The payload [Codec] is swappable via [WithCodec] and
// defaults to encoding/json/v2 ([DefaultCodec]). Faster opt-in codecs live in
// the codec/sonic and codec/goccy subpackages, which carry their own module
// dependencies and never enter this package's module graph.
//
// # Serving
//
// A [Conn] is a symmetric peer that can both issue ([Conn.Call], [Conn.Notify])
// and answer requests via a [Handler] started with [Conn.Go]. For network
// servers, [Serve] and [ListenAndServe] accept connections from a
// [net.Listener] and drive each one with a [StreamServer], typically built from
// a [Handler] with [HandlerServer].
package jsonrpc2
