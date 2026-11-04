// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "go.lsp.dev/jsonrpc2"

const (
	// LSPReservedErrorRangeStart is the start range of LSP reserved error codes.
	//
	// It doesn't denote a real error code.
	LSPReservedErrorRangeStart jsonrpc2.Code = -32899

	// CodeContentModified is the state change that invalidates the result of a request in execution.
	//
	// Defined by the protocol.
	CodeContentModified jsonrpc2.Code = -32801

	// CodeRequestCancelled is the cancellation error.
	//
	// Defined by the protocol.
	CodeRequestCancelled jsonrpc2.Code = -32800

	// LSPReservedErrorRangeEnd is the end range of LSP reserved error codes.
	//
	// It doesn't denote a real error code.
	LSPReservedErrorRangeEnd jsonrpc2.Code = -32800
)

var (
	// ErrContentModified should be used when a request's result is invalidated by
	// a change to the document or workspace before the request completes.
	ErrContentModified = jsonrpc2.NewError(CodeContentModified, "content modified")

	// ErrRequestCancelled should be used when a request is canceled early.
	ErrRequestCancelled = jsonrpc2.NewError(CodeRequestCancelled, "cancelled JSON-RPC")
)
