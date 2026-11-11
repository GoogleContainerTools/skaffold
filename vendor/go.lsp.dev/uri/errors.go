// Copyright 2026 The Go Language Server Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

import (
	"errors"
	"fmt"
)

var (
	// ErrMissingScheme reports that strict URI parsing found no scheme.
	ErrMissingScheme = errors.New("uri: scheme is missing")
	// ErrInvalidScheme reports that a URI scheme contains illegal characters.
	ErrInvalidScheme = errors.New("uri: scheme contains illegal characters")
	// ErrAuthorityPath reports that an authority URI path is not empty or slash-prefixed.
	ErrAuthorityPath = errors.New("uri: authority path must be empty or begin with slash")
	// ErrPathAuthority reports that a URI path without authority starts with two slashes.
	ErrPathAuthority = errors.New("uri: path without authority cannot begin with two slashes")
)

// Error describes a URI validation failure while preserving a typed cause.
type Error struct {
	Op    string
	Input string
	Err   error
}

// Error returns a human-readable URI error string.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Input == "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("%s %q: %v", e.Op, e.Input, e.Err)
}

// Unwrap returns the underlying sentinel error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func uriError(op, input string, err error) error {
	return &Error{Op: op, Input: input, Err: err}
}
