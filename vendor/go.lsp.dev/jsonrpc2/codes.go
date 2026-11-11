// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

// Code is an error code as defined by the JSON-RPC 2.0 specification.
//
// See https://www.jsonrpc.org/specification#error_object for details.
type Code int32

// Standard JSON-RPC 2.0 error codes and the LSP-reserved range.
const (
	// ParseError indicates that invalid JSON was received by the server, or that
	// an error occurred on the server while parsing the JSON text.
	ParseError Code = -32700

	// InvalidRequest indicates that the JSON sent is not a valid Request object.
	InvalidRequest Code = -32600

	// MethodNotFound indicates that the method does not exist or is not
	// available.
	MethodNotFound Code = -32601

	// InvalidParams indicates invalid method parameter(s).
	InvalidParams Code = -32602

	// InternalError is the internal JSON-RPC error.
	InternalError Code = -32603

	// JSONRPCReservedErrorRangeStart is the start of the JSON-RPC reserved error
	// code range.
	//
	// It does not denote a real error code. No LSP error codes should be defined
	// between the start and end of the range. For backwards compatibility the
	// ServerNotInitialized and UnknownError codes are left in the range.
	//
	// @since 3.16.0.
	JSONRPCReservedErrorRangeStart Code = -32099

	// CodeServerErrorStart is reserved for implementation-defined server errors.
	//
	// Deprecated: Use JSONRPCReservedErrorRangeStart instead.
	CodeServerErrorStart = JSONRPCReservedErrorRangeStart

	// ServerNotInitialized indicates that the server has not been initialized.
	ServerNotInitialized Code = -32002

	// UnknownError should be used for all non-coded errors.
	UnknownError Code = -32001

	// JSONRPCReservedErrorRangeEnd is the end of the JSON-RPC reserved error code
	// range.
	//
	// It does not denote a real error code.
	//
	// @since 3.16.0.
	JSONRPCReservedErrorRangeEnd Code = -32000

	// CodeServerErrorEnd is reserved for implementation-defined server errors.
	//
	// Deprecated: Use JSONRPCReservedErrorRangeEnd instead.
	CodeServerErrorEnd = JSONRPCReservedErrorRangeEnd
)
