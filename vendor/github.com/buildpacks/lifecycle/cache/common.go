package cache

import (
	"errors"
)

var errCacheCommitted = errors.New("cache cannot be modified after commit")

// ReadErr is an error type for filesystem read errors.
type ReadErr struct {
	msg string
}

// NewReadErr creates a new ReadErr.
func NewReadErr(msg string) ReadErr {
	return ReadErr{msg: msg}
}

// Error returns the error message.
func (e ReadErr) Error() string {
	return e.msg
}

// IsReadErr checks if an error is a ReadErr.
func IsReadErr(err error) (bool, *ReadErr) {
	var e ReadErr
	isReadErr := errors.As(err, &e)
	return isReadErr, &e
}
