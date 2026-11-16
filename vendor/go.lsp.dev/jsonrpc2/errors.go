// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"errors"
	"fmt"
	"strconv"
)

// Error is the wire representation of a JSON-RPC error object.
//
// It is the structured value carried in the "error" member of a [Response]. It
// implements the error interface so that a handler can return it directly, and
// it preserves its [Code] across the error<->wire mapping performed by
// [toWireError].
type Error struct {
	// Message is a short description of the error.
	Message string `json:"message"`

	// Data is an optional primitive or structured value that carries additional
	// information about the error. It is omitted from the wire form when the
	// zero value.
	Data RawMessage `json:"data,omitzero"`

	// Code is a number indicating the error type that occurred.
	Code Code `json:"code"`
}

// compile-time check that *Error implements error.
var _ error = (*Error)(nil)

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Is reports whether the target error is an [*Error] with the same [Code]. This
// lets callers match against the sentinel errors with errors.Is.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}

// NewError builds an [*Error] for the supplied code and message.
func NewError(c Code, message string) *Error {
	return &Error{
		Code:    c,
		Message: message,
	}
}

// Errorf builds an [*Error] for the supplied code, using the format specifier
// and arguments to construct the message.
func Errorf(c Code, format string, args ...any) *Error {
	return &Error{
		Code:    c,
		Message: fmt.Sprintf(format, args...),
	}
}

// Standard JSON-RPC 2.0 errors, exposed as sentinels for use with errors.Is.
var (
	// ErrUnknown should be used for all non-coded errors.
	ErrUnknown = NewError(UnknownError, "JSON-RPC unknown error")

	// ErrParse is used when invalid JSON was received by the server.
	ErrParse = NewError(ParseError, "JSON-RPC parse error")

	// ErrInvalidRequest is used when the JSON sent is not a valid Request object.
	ErrInvalidRequest = NewError(InvalidRequest, "JSON-RPC invalid request")

	// ErrMethodNotFound should be returned by the handler when the method does
	// not exist or is not available.
	ErrMethodNotFound = NewError(MethodNotFound, "JSON-RPC method not found")

	// ErrInvalidParams should be returned by the handler when the method
	// parameter(s) were invalid.
	ErrInvalidParams = NewError(InvalidParams, "JSON-RPC invalid params")

	// ErrInternal indicates a failure to process a call correctly.
	ErrInternal = NewError(InternalError, "JSON-RPC internal error")
)

// constError is a comparable, immutable error backed by a string constant.
type constError string

// compile-time check that constError implements error.
var _ error = constError("")

// Error implements the error interface.
func (e constError) Error() string { return string(e) }

const (
	// ErrIdleTimeout is returned when serving timed out waiting for new
	// connections.
	ErrIdleTimeout = constError("timed out waiting for new connections")

	// ErrClientClosing is returned for an outgoing [Conn.Call] or [Conn.Notify]
	// once the connection is shutting down: the local end has called Close, or
	// the read or write side of the stream has failed.
	ErrClientClosing = constError("jsonrpc2: client is closing")

	// ErrServerClosing is the error written in response to an incoming call that
	// arrives, or is still queued, once the connection is shutting down.
	ErrServerClosing = constError("jsonrpc2: server is closing")
)

// toWireError maps an arbitrary error onto the [*Error] that should be written
// on the wire, preserving the [Code] of any wrapped [*Error].
//
// A nil error maps to nil (the response is a success). An error that already is
// an [*Error] is returned unchanged. Otherwise the outer error's message is kept
// and, if it wraps an [*Error], that wrapped error's code is preserved.
func toWireError(err error) *Error {
	if err == nil {
		return nil
	}

	var wire *Error
	if errors.As(err, &wire) && wire.Error() == err.Error() {
		// The error is (or unwraps directly to) a wire error with the same
		// message; use it verbatim so its code and data survive.
		return wire
	}

	result := &Error{Message: err.Error()}
	if errors.As(err, &wire) {
		// Outer message, inner code: a handler may wrap a coded error with extra
		// context; keep the code so the peer still sees the classification.
		result.Code = wire.Code
		result.Data = wire.Data
	}
	return result
}

// appendError appends the JSON encoding of the error object to dst:
//
//	{"code":<int>,"message":<esc>,"data":<raw>}
//
// The data member is omitted when e.Data carries no bytes (nil or empty), so an
// empty-but-non-nil [RawMessage] cannot emit an invalid bare "data": member.
func appendError(dst []byte, e *Error) []byte {
	dst = append(dst, `{"code":`...)
	dst = strconv.AppendInt(dst, int64(e.Code), 10)
	dst = append(dst, `,"message":`...)
	dst = appendQuotedString(dst, e.Message)
	if len(e.Data) > 0 {
		dst = append(dst, `,"data":`...)
		dst = append(dst, e.Data...)
	}
	return append(dst, '}')
}

// decodeError decodes an error object span into an [*Error]. It reuses the
// shared span scanner so the same nesting/escape handling applies to the nested
// object, and copies the data span so the result owns its bytes.
//
// It returns ok=false when the span is not a well-formed error object.
func decodeError(span []byte) (e *Error, ok bool) {
	var codeSpan, msgSpan, dataSpan []byte
	if !scanErrorObject(span, &codeSpan, &msgSpan, &dataSpan) {
		return nil, false
	}

	out := new(Error)
	if codeSpan != nil {
		n, perr := strconv.ParseInt(string(codeSpan), 10, 32)
		if perr != nil {
			return nil, false
		}
		out.Code = Code(n)
	}
	if msgSpan != nil {
		s, sok := unquoteJSONString(msgSpan)
		if !sok {
			return nil, false
		}
		out.Message = s
	}
	if dataSpan != nil {
		out.Data = RawMessage(cloneBytes(dataSpan))
	}
	return out, true
}
