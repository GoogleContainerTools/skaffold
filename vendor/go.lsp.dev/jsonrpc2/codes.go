// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

// Code is an error code as defined in the JSON-RPC spec.
type Code int32

// list of JSON-RPC error codes.
const (
	// ParseError is the invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ParseError Code = -32700

	// InvalidRequest is the JSON sent is not a valid Request object.
	InvalidRequest Code = -32600

	// MethodNotFound is the method does not exist / is not available.
	MethodNotFound Code = -32601

	// InvalidParams is the invalid method parameter(s).
	InvalidParams Code = -32602

	// InternalError is the internal JSON-RPC error.
	InternalError Code = -32603

	// JSONRPCReservedErrorRangeStart is the start range of JSON RPC reserved error codes.
	//
	// It doesn't denote a real error code. No LSP error codes should
	// be defined between the start and end range. For backwards
	// compatibility the "ServerNotInitialized" and the "UnknownErrorCode"
	// are left in the range.
	//
	// @since 3.16.0.
	JSONRPCReservedErrorRangeStart Code = -32099

	// CodeServerErrorStart reserved for implementation-defined server-errors.
	//
	// Deprecated: Use JSONRPCReservedErrorRangeStart instead.
	CodeServerErrorStart = JSONRPCReservedErrorRangeStart

	// ServerNotInitialized is the error of server not initialized.
	ServerNotInitialized Code = -32002

	// UnknownError should be used for all non coded errors.
	UnknownError Code = -32001

	// JSONRPCReservedErrorRangeEnd is the start range of JSON RPC reserved error codes.
	//
	// It doesn't denote a real error code.
	//
	// @since 3.16.0.
	JSONRPCReservedErrorRangeEnd Code = -32000

	// CodeServerErrorEnd reserved for implementation-defined server-errors.
	//
	// Deprecated: Use JSONRPCReservedErrorRangeEnd instead.
	CodeServerErrorEnd = JSONRPCReservedErrorRangeEnd
)

// This file contains the Go forms of the wire specification.
//
// See http://www.jsonrpc.org/specification for details.
//
// list of JSON-RPC errors.
var (
	// ErrUnknown should be used for all non coded errors.
	ErrUnknown = NewError(UnknownError, "JSON-RPC unknown error")

	// ErrParse is used when invalid JSON was received by the server.
	ErrParse = NewError(ParseError, "JSON-RPC parse error")

	// ErrInvalidRequest is used when the JSON sent is not a valid Request object.
	ErrInvalidRequest = NewError(InvalidRequest, "JSON-RPC invalid request")

	// ErrMethodNotFound should be returned by the handler when the method does
	// not exist / is not available.
	ErrMethodNotFound = NewError(MethodNotFound, "JSON-RPC method not found")

	// ErrInvalidParams should be returned by the handler when method
	// parameter(s) were invalid.
	ErrInvalidParams = NewError(InvalidParams, "JSON-RPC invalid params")

	// ErrInternal is not currently returned but defined for completeness.
	ErrInternal = NewError(InternalError, "JSON-RPC internal error")
)
