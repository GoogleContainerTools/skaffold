/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	stderrors "errors"

	pkgerrors "github.com/pkg/errors"
)

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) error {
	return pkgerrors.New(message)
}

// NewWithoutStack is like new but does NOT wrap with a stack
// This is useful for exported errors
func NewWithoutStack(message string) error {
	return stderrors.New(message)
}

// Errorf formats according to a format specifier and returns the string as a
// value that satisfies error. Errorf also records the stack trace at the
// point it was called.
func Errorf(format string, args ...interface{}) error {
	return pkgerrors.Errorf(format, args...)
}

// Wrap returns an error annotating err with a stack trace at the point Wrap
// is called, and the supplied message. If err is nil, Wrap returns nil.
func Wrap(err error, message string) error {
	return pkgerrors.Wrap(err, message)
}

// Wrapf returns an error annotating err with a stack trace at the point Wrapf
// is called, and the format specifier. If err is nil, Wrapf returns nil.
func Wrapf(err error, format string, args ...interface{}) error {
	return pkgerrors.Wrapf(err, format, args...)
}

// WithStack annotates err with a stack trace at the point WithStack was called.
// If err is nil, WithStack returns nil.
func WithStack(err error) error {
	return pkgerrors.WithStack(err)
}

// Causer is an interface to github.com/pkg/errors error's Cause() wrapping
type Causer interface {
	// Cause returns the underlying error
	Cause() error
}

// StackTracer is an interface to github.com/pkg/errors error's StackTrace()
type StackTracer interface {
	// StackTrace returns the StackTrace ...
	// TODO: return our own type instead?
	// https://github.com/pkg/errors#roadmap
	StackTrace() pkgerrors.StackTrace
}

// StackTrace returns the deepest StackTrace in a Cause chain
// https://github.com/pkg/errors/issues/173
func StackTrace(err error) pkgerrors.StackTrace {
	var stackErr error
	for {
		if _, ok := err.(StackTracer); ok {
			stackErr = err
		}
		if causerErr, ok := err.(Causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}
	if stackErr != nil {
		return stackErr.(StackTracer).StackTrace()
	}
	return nil
}
