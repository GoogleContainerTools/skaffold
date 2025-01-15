package build

import (
	"errors"
	"fmt"
	"log"

	"github.com/mmcloughlin/avo/internal/stack"
	"github.com/mmcloughlin/avo/src"
)

// Error represents an error during building, optionally tagged with the position at which it happened.
type Error struct {
	Position src.Position
	Err      error
}

// exterr constructs an Error with position derived from the first frame in the
// call stack outside this package.
func exterr(err error) Error {
	e := Error{Err: err}
	if f := stack.ExternalCaller(); f != nil {
		e.Position = src.FramePosition(*f).Relwd()
	}
	return e
}

func (e Error) Error() string {
	msg := e.Err.Error()
	if e.Position.IsValid() {
		return e.Position.String() + ": " + msg
	}
	return msg
}

// ErrorList is a collection of errors for a source file.
type ErrorList []Error

// Add appends an error to the list.
func (e *ErrorList) Add(err Error) {
	*e = append(*e, err)
}

// AddAt appends an error at position p.
func (e *ErrorList) AddAt(p src.Position, err error) {
	e.Add(Error{p, err})
}

// addext appends an error to the list, tagged with the first external position
// outside this package.
func (e *ErrorList) addext(err error) {
	e.Add(exterr(err))
}

// Err returns an error equivalent to this error list.
// If the list is empty, Err returns nil.
func (e ErrorList) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

// Error implements the error interface.
func (e ErrorList) Error() string {
	switch len(e) {
	case 0:
		return "no errors"
	case 1:
		return e[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", e[0], len(e)-1)
}

// LogError logs a list of errors, one error per line, if the err parameter is
// an ErrorList. Otherwise it just logs the err string. Reports at most max
// errors, or unlimited if max is 0.
func LogError(l *log.Logger, err error, max int) {
	var list ErrorList
	if errors.As(err, &list) {
		for i, e := range list {
			if max > 0 && i == max {
				l.Print("too many errors")
				return
			}
			l.Printf("%s\n", e)
		}
	} else if err != nil {
		l.Printf("%s\n", err)
	}
}
