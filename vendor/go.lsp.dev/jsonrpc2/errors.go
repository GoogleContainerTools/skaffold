// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"errors"
	"fmt"

	"github.com/segmentio/encoding/json"
)

// Error represents a JSON-RPC error.
type Error struct {
	// Code a number indicating the error type that occurred.
	Code Code `json:"code"`

	// Message a string providing a short description of the error.
	Message string `json:"message"`

	// Data a Primitive or Structured value that contains additional
	// information about the error. Can be omitted.
	Data *json.RawMessage `json:"data,omitempty"`
}

// compile time check whether the Error implements error interface.
var _ error = (*Error)(nil)

// Error implements error.Error.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap implements errors.Unwrap.
//
// Returns the error underlying the receiver, which may be nil.
func (e *Error) Unwrap() error { return errors.New(e.Message) }

// NewError builds a Error struct for the suppied code and message.
func NewError(c Code, message string) *Error {
	return &Error{
		Code:    c,
		Message: message,
	}
}

// Errorf builds a Error struct for the suppied code, format and args.
func Errorf(c Code, format string, args ...interface{}) *Error {
	return &Error{
		Code:    c,
		Message: fmt.Sprintf(format, args...),
	}
}

// constErr represents a error constant.
type constErr string

// compile time check whether the constErr implements error interface.
var _ error = (*constErr)(nil)

// Error implements error.Error.
func (e constErr) Error() string { return string(e) }

const (
	// ErrIdleTimeout is returned when serving timed out waiting for new connections.
	ErrIdleTimeout = constErr("timed out waiting for new connections")
)
